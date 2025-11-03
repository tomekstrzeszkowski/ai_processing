import { createContext, useContext } from 'react';

const ProtocolContext = createContext<WebSocketContextType | null>(null);

export const useProtocol = () => {
  const context = useContext(ProtocolContext);
  if (!context) {
    throw new Error('useProtocol must be used within a ProtocolContext');
  }
  return context;
};

export const ProtocolProvider = ({ children }: { children: React.ReactNode }) => {
  return (
    <ProtocolContext.Provider value={value}>
      {children}
    </ProtocolContext.Provider>
  );
};
export default ProtocolProvider;