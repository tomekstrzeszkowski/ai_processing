import { createContext, useContext } from 'react';

// Will be used for app runnin in internal home network
type HttpContextType = {
};
const HttpContext = createContext<HttpContextType | null>(null);

export const useHttp = () => {
  const context = useContext(HttpContext);
  if (!context) {
    throw new Error('useHttp must be used within a HttpContext');
  }
  return context;
};

export const HttpProvider = ({ children }: { children: React.ReactNode }) => {

  return (
    <HttpContext.Provider value={{ 

    }}>
      {children}
    </HttpContext.Provider>
  );
};
export default HttpProvider;