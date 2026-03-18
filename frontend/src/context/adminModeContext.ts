import { createContext } from 'react';

export interface AdminModeContextValue {
  isAdminMode: boolean;
  setIsAdminMode: (value: boolean) => void;
}

export const AdminModeContext = createContext<AdminModeContextValue | undefined>(undefined);
