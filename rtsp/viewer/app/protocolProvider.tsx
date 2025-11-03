import { createContext, useContext, useRef } from 'react';
type ProtocolContextType = {
  protocol: React.RefObject<string | null>;
};
const ProtocolContext = createContext<ProtocolContextType | null>(null);

export const useProtocol = () => {
  const context = useContext(ProtocolContext);
  if (!context) {
    throw new Error('useProtocol must be used within a ProtocolContext');
  }
  return context;
};

export const ProtocolProvider = ({ children }: { children: React.ReactNode }) => {
  const protocol = useRef<string>("WEBRTC_PROTOCOL");
  const value = { protocol };
  return (
    <ProtocolContext.Provider value={value}>
      {children}
    </ProtocolContext.Provider>
  );
};
export default ProtocolProvider;