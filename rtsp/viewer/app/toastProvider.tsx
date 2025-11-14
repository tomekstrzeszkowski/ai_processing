import { createContext, useContext } from 'react';
import {
    Alert,
    Platform
} from 'react-native';


const showAlert = (title: string, message: string="") => {
  if (Platform.OS === 'web') {
    alert(`${title}: ${message}`);
  } else {
    Alert.alert(title, message);
  }
};

type ToastContextType = {
    showAlert: (title: string, message?: string) => void
};
const ToastContext = createContext<ToastContextType | null>(null);

export const useToast = () => {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error('useToast must be used within a ToastContext');
  }
  return context;
};

export const ToastProvider = ({ children }: { children: React.ReactNode }) => {

  return (
    <ToastContext.Provider value={{ 
        showAlert
    }}>
      {children}
    </ToastContext.Provider>
  );
};
export default ToastProvider;