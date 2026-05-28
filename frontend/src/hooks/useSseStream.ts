import { useCallback, useEffect, useRef, useState } from 'react';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '';

const DEFAULT_MIN_BACKOFF_MS = 1_000;
const DEFAULT_MAX_BACKOFF_MS = 30_000;
const DEFAULT_HEARTBEAT_TIMEOUT_MS = 15_000;
const DEFAULT_HEARTBEAT_CHECK_INTERVAL_MS = 5_000;

interface SseEvent {
  event?: string;
  data: string;
  id?: string;
  retry?: number;
}

export interface UseSseStreamOptions<TState, TEvent> {
  url: string;
  getInitialState: () => TState;
  parseMessage: (data: string) => TEvent | null;
  reduceState: (state: TState, event: TEvent) => TState;
  resetStateOnConnect?: boolean;
  minBackoffMs?: number;
  maxBackoffMs?: number;
  heartbeatTimeoutMs?: number;
  heartbeatCheckIntervalMs?: number;
  requestInit?: Omit<RequestInit, 'signal'>;
}

export interface UseSseStreamState<TState> {
  data: TState;
  isConnected: boolean;
  error: string | null;
  retry: () => void;
}

function parseSseEvent(block: string): SseEvent | null {
  const lines = block.split('\n');
  const event: SseEvent = { data: '' };
  const dataLines: string[] = [];
  let hasField = false;

  for (const line of lines) {
    if (line === '' || line.startsWith(':')) {
      continue;
    }

    const colonIndex = line.indexOf(':');
    const field = colonIndex === -1 ? line : line.substring(0, colonIndex);
    let value = colonIndex === -1 ? '' : line.substring(colonIndex + 1);

    if (value.startsWith(' ')) {
      value = value.substring(1);
    }

    hasField = true;

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
      case 'retry': {
        const retryMs = parseInt(value, 10);

        if (!Number.isNaN(retryMs)) {
          event.retry = retryMs;
        }

        break;
      }
    }
  }

  if (!hasField) {
    return null;
  }

  if (dataLines.length > 0) {
    event.data = dataLines.join('\n');
  }

  return event;
}

function normalizeSseChunk(chunk: string): string {
  return chunk.replace(/\r\n/g, '\n').replace(/\r/g, '\n');
}

function createHeaders(headers?: HeadersInit): Headers {
  const result = new Headers(headers);

  if (!result.has('Accept')) {
    result.set('Accept', 'text/event-stream');
  }

  return result;
}

function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error) {
    return error.message;
  }

  return fallback;
}

function resolveStreamUrl(url: string): string {
  if (url.startsWith('http://') || url.startsWith('https://')) {
    return url;
  }

  return `${API_BASE_URL}${url}`;
}

export function useSseStream<TState, TEvent>({
  url,
  getInitialState,
  parseMessage,
  reduceState,
  resetStateOnConnect = false,
  minBackoffMs = DEFAULT_MIN_BACKOFF_MS,
  maxBackoffMs = DEFAULT_MAX_BACKOFF_MS,
  heartbeatTimeoutMs = DEFAULT_HEARTBEAT_TIMEOUT_MS,
  heartbeatCheckIntervalMs = DEFAULT_HEARTBEAT_CHECK_INTERVAL_MS,
  requestInit,
}: UseSseStreamOptions<TState, TEvent>): UseSseStreamState<TState> {
  const [data, setData] = useState<TState>(() => getInitialState());
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const abortControllerRef = useRef<AbortController | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const heartbeatCheckTimerRef = useRef<ReturnType<typeof setInterval> | undefined>(undefined);
  const backoffRef = useRef(minBackoffMs);
  const lastActivityRef = useRef<number>(Date.now());
  const connectRef = useRef<() => void>(() => undefined);
  const getInitialStateRef = useRef(getInitialState);
  const parseMessageRef = useRef(parseMessage);
  const reduceStateRef = useRef(reduceState);
  const requestInitRef = useRef(requestInit);

  getInitialStateRef.current = getInitialState;
  parseMessageRef.current = parseMessage;
  reduceStateRef.current = reduceState;
  requestInitRef.current = requestInit;

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
    reconnectTimerRef.current = setTimeout(() => {
      connectRef.current();
    }, delay);

    backoffRef.current = Math.min(backoffRef.current * 2, maxBackoffMs);
  }, [clearReconnectTimer, maxBackoffMs]);

  const connect = useCallback(() => {
    clearReconnectTimer();
    clearHeartbeatCheckTimer();

    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }

    const abortController = new AbortController();
    abortControllerRef.current = abortController;
    lastActivityRef.current = Date.now();

    heartbeatCheckTimerRef.current = setInterval(() => {
      const timeSinceLastActivity = Date.now() - lastActivityRef.current;

      if (timeSinceLastActivity > heartbeatTimeoutMs) {
        setIsConnected(false);
        setError('Connection timeout. Reconnecting...');

        if (abortControllerRef.current) {
          abortControllerRef.current.abort();
          abortControllerRef.current = null;
        }

        clearHeartbeatCheckTimer();
        scheduleReconnect();
      }
    }, heartbeatCheckIntervalMs);

    void (async () => {
      try {
        const currentRequestInit = requestInitRef.current;
        const response = await fetch(resolveStreamUrl(url), {
          ...currentRequestInit,
          signal: abortController.signal,
          headers: createHeaders(currentRequestInit?.headers),
        });

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        if (!response.body) {
          throw new Error('Response body is null');
        }

        setIsConnected(true);
        setError(null);

        if (resetStateOnConnect) {
          setData(getInitialStateRef.current());
        }

        backoffRef.current = minBackoffMs;

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';

        while (true) {
          const { done, value } = await reader.read();

          if (done) {
            break;
          }

          lastActivityRef.current = Date.now();
          buffer = normalizeSseChunk(buffer + decoder.decode(value, { stream: true }));

          let eventEndIndex = buffer.indexOf('\n\n');
          while (eventEndIndex !== -1) {
            const eventBlock = buffer.substring(0, eventEndIndex);
            buffer = buffer.substring(eventEndIndex + 2);

            if (eventBlock.trim() !== '') {
              const sseEvent = parseSseEvent(eventBlock);

              if (sseEvent !== null) {
                if (sseEvent.event === 'error') {
                  setError(sseEvent.data || 'Stream error');
                } else {
                  try {
                    const parsedEvent = parseMessageRef.current(sseEvent.data);

                    if (parsedEvent !== null) {
                      setData((previousState) => reduceStateRef.current(previousState, parsedEvent));
                    }
                  } catch (streamError) {
                    setError(getErrorMessage(streamError, 'Failed to process stream event'));
                  }
                }
              }
            }

            eventEndIndex = buffer.indexOf('\n\n');
          }
        }

        setIsConnected(false);
        clearHeartbeatCheckTimer();
        scheduleReconnect();
      } catch (streamError) {
        if (streamError instanceof Error && streamError.name === 'AbortError') {
          return;
        }

        setIsConnected(false);
        setError(getErrorMessage(streamError, 'Connection error. Reconnecting...'));

        clearHeartbeatCheckTimer();
        scheduleReconnect();
      }
    })();
  }, [
    clearHeartbeatCheckTimer,
    clearReconnectTimer,
    heartbeatCheckIntervalMs,
    heartbeatTimeoutMs,
    minBackoffMs,
    resetStateOnConnect,
    scheduleReconnect,
    url,
  ]);

  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);

  const retry = useCallback(() => {
    backoffRef.current = minBackoffMs;
    connect();
  }, [connect, minBackoffMs]);

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
  }, [clearHeartbeatCheckTimer, clearReconnectTimer, connect]);

  return {
    data,
    isConnected,
    error,
    retry,
  };
}
