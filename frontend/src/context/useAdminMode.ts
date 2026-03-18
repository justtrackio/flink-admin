import { useContext } from 'react';
import type { AdminModeContextValue } from './adminModeContext';
import { AdminModeContext } from './adminModeContext';

export function useAdminMode(): AdminModeContextValue {
  const context = useContext(AdminModeContext);

  if (!context) {
    throw new Error('useAdminMode must be used within AdminModeProvider');
  }

  return context;
}
