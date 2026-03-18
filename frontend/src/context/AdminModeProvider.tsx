import type { ReactNode } from 'react';
import { useMemo, useState } from 'react';
import { AdminModeContext } from './adminModeContext';

interface AdminModeProviderProps {
  children: ReactNode;
}

export function AdminModeProvider({ children }: AdminModeProviderProps) {
  const [isAdminMode, setIsAdminMode] = useState(false);
  const value = useMemo(() => ({ isAdminMode, setIsAdminMode }), [isAdminMode]);

  return <AdminModeContext.Provider value={value}>{children}</AdminModeContext.Provider>;
}
