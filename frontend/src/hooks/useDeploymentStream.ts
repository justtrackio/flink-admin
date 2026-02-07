import { useEffect, useState, useRef, useCallback } from 'react';
import type { FlinkDeployment, DeploymentEvent } from '../api/schema';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '';

// Reconnection backoff configuration
const MIN_BACKOFF_MS = 1_000; // 1 second
const MAX_BACKOFF_MS = 30_000; // 30 seconds

// Heartbeat timeout: if we don't receive ANY data (including heartbeat comments)
// within this period, assume the connection is dead and reconnect
// Server sends heartbeats every 5s, so 15s = 3 missed heartbeats
const HEARTBEAT_TIMEOUT_MS = 15_000;

// How often to check for heartbeat timeout
const HEARTBEAT_CHECK_INTERVAL_MS = 5_000;

interface DeploymentStreamState {
  deployments: FlinkDeployment[];
  isConnected: boolean;
  error: string | null;
  retry: () => void;
}

interface SSEEvent {
  event?: string;
  data: string;
  id?: string;
  retry?: number;
}

/**
 * Parses a complete SSE event block (between \n\n delimiters).
 * Handles multi-line data fields and comment lines (starting with :).
 */
function parseSSEEvent(block: string): SSEEvent | null {
  const lines = block.split('\n');
  
  // Check if it's a comment (heartbeat)
  if (lines.length === 1 && lines[0].startsWith(':')) {
    return null; // Comments don't produce events
  }

  const event: SSEEvent = { data: '' };
  const dataLines: string[] = [];

  for (const line of lines) {
    if (line.startsWith(':')) {
      // Comment line, skip
      continue;
    }

    const colonIndex = line.indexOf(':');
    if (colonIndex === -1) {
      continue; // Invalid line, skip
    }

    const field = line.substring(0, colonIndex);
    // Value starts after colon, skip optional space
    let value = line.substring(colonIndex + 1);
    if (value.startsWith(' ')) {
      value = value.substring(1);
    }

    switch (field) {
      case 'event':
        event.event = value;
        break;
      case 'data':
        dataLines.push(value);
        break;
      case 'id':
        event.id = value;
        break;
      case 'retry':
        const retryMs = parseInt(value, 10);
        if (!isNaN(retryMs)) {
          event.retry = retryMs;
        }
        break;
    }
  }

  // Join multi-line data with newlines
  if (dataLines.length > 0) {
    event.data = dataLines.join('\n');
  }

  return event;
}

export function useDeploymentStream(): DeploymentStreamState {
  const [deploymentsMap, setDeploymentsMap] = useState<Map<string, FlinkDeployment>>(new Map());
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const abortControllerRef = useRef<AbortController | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const heartbeatCheckTimerRef = useRef<ReturnType<typeof setInterval> | undefined>(undefined);
  const backoffRef = useRef(MIN_BACKOFF_MS);
  const lastActivityRef = useRef<number>(Date.now());

  const clearReconnectTimer = useCallback(() => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = undefined;
    }
  }, []);

  const clearHeartbeatCheckTimer = useCallback(() => {
    if (heartbeatCheckTimerRef.current) {
      clearInterval(heartbeatCheckTimerRef.current);
      heartbeatCheckTimerRef.current = undefined;
    }
  }, []);

  const scheduleReconnect = useCallback(() => {
    clearReconnectTimer();
    
    const delay = backoffRef.current;
    console.log(`Scheduling reconnect in ${delay}ms`);
    
    reconnectTimerRef.current = setTimeout(() => {
      connect();
    }, delay);
    
    // Exponential backoff with max cap
    backoffRef.current = Math.min(backoffRef.current * 2, MAX_BACKOFF_MS);
  }, [clearReconnectTimer]);

  const connect = useCallback(() => {
    // Clean up existing connection
    clearReconnectTimer();
    clearHeartbeatCheckTimer();
    
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }

    const abortController = new AbortController();
    abortControllerRef.current = abortController;
    lastActivityRef.current = Date.now();

    const url = `${API_BASE_URL}/api/deployments/watch`;
    
    // Start heartbeat timeout checker
    heartbeatCheckTimerRef.current = setInterval(() => {
      const timeSinceLastActivity = Date.now() - lastActivityRef.current;
      
      if (timeSinceLastActivity > HEARTBEAT_TIMEOUT_MS) {
        console.warn(`No data received for ${timeSinceLastActivity}ms (timeout: ${HEARTBEAT_TIMEOUT_MS}ms). Reconnecting...`);
        setIsConnected(false);
        setError('Connection timeout. Reconnecting...');
        
        // Abort the fetch and trigger reconnect
        if (abortControllerRef.current) {
          abortControllerRef.current.abort();
          abortControllerRef.current = null;
        }
        
        clearHeartbeatCheckTimer();
        scheduleReconnect();
      }
    }, HEARTBEAT_CHECK_INTERVAL_MS);

    (async () => {
      try {
        const response = await fetch(url, {
          signal: abortController.signal,
          headers: {
            'Accept': 'text/event-stream',
          },
        });

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        if (!response.body) {
          throw new Error('Response body is null');
        }

        console.log('SSE connection opened');
        setIsConnected(true);
        setError(null);
        
        // Clear stale state: the server will send fresh ADDED events for all current deployments
        setDeploymentsMap(new Map());
        
        // Reset backoff on successful connection
        backoffRef.current = MIN_BACKOFF_MS;

        // Read the stream
        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';

        while (true) {
          const { done, value } = await reader.read();
          
          if (done) {
            console.log('SSE stream ended');
            break;
          }

          // Update activity timestamp on every chunk received (including heartbeats)
          lastActivityRef.current = Date.now();

          // Decode chunk and add to buffer
          buffer += decoder.decode(value, { stream: true });

          // Process complete events (delimited by \n\n)
          let eventEndIndex: number;
          while ((eventEndIndex = buffer.indexOf('\n\n')) !== -1) {
            const eventBlock = buffer.substring(0, eventEndIndex);
            buffer = buffer.substring(eventEndIndex + 2);

            if (eventBlock.trim() === '') {
              continue; // Empty block, skip
            }

            const sseEvent = parseSSEEvent(eventBlock);
            
            if (sseEvent === null) {
              // Comment/heartbeat, already tracked via lastActivityRef
              continue;
            }

            // Handle error events
            if (sseEvent.event === 'error') {
              console.error('SSE error event:', sseEvent.data);
              setError(sseEvent.data);
              continue;
            }

            // Handle data events
            if (sseEvent.data) {
              try {
                const deploymentEvent: DeploymentEvent = JSON.parse(sseEvent.data);
                const { type, deployment } = deploymentEvent;

                setDeploymentsMap((prev) => {
                  const next = new Map(prev);
                  
                  if (type === 'DELETED') {
                    next.delete(deployment.metadata.uid);
                  } else {
                    // ADDED or MODIFIED
                    next.set(deployment.metadata.uid, deployment);
                  }
                  
                  return next;
                });
              } catch (err) {
                console.error('Failed to parse SSE event data:', err);
                setError(err instanceof Error ? err.message : 'Failed to parse event');
              }
            }
          }
        }

        // Stream ended normally, reconnect
        console.log('Stream ended, reconnecting...');
        setIsConnected(false);
        clearHeartbeatCheckTimer();
        scheduleReconnect();

      } catch (err) {
        // Check if it was aborted (intentional disconnect)
        if (err instanceof Error && err.name === 'AbortError') {
          console.log('Fetch aborted');
          return;
        }

        console.error('SSE connection error:', err);
        setIsConnected(false);
        setError(err instanceof Error ? err.message : 'Connection error. Reconnecting...');
        
        clearHeartbeatCheckTimer();
        scheduleReconnect();
      }
    })();
  }, [clearReconnectTimer, clearHeartbeatCheckTimer, scheduleReconnect]);

  const retry = useCallback(() => {
    console.log('Manual retry triggered');
    backoffRef.current = MIN_BACKOFF_MS;
    connect();
  }, [connect]);

  useEffect(() => {
    connect();

    return () => {
      clearReconnectTimer();
      clearHeartbeatCheckTimer();
      
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
        abortControllerRef.current = null;
      }
    };
  }, [connect, clearReconnectTimer, clearHeartbeatCheckTimer]);

  return {
    deployments: Array.from(deploymentsMap.values()),
    isConnected,
    error,
    retry,
  };
}
