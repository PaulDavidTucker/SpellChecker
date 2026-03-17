import { useState, useEffect, useCallback } from 'react';

interface ServerStatus {
  ready: boolean;
  loadTime?: number;
  message?: string;
}

export function useServerReady() {
  const [status, setStatus] = useState<ServerStatus>({
    ready: false,
    message: 'Connecting...',
  });
  const [isChecking, setIsChecking] = useState(true);

  const checkReady = useCallback(async () => {
    try {
      const response = await fetch('/health');
      const data = await response.json();
      setStatus({
        ready: data.ready,
        loadTime: data.load_time,
        message: data.message,
      });
    } catch (error) {
      setStatus({
        ready: false,
        message: 'Server unreachable',
      });
    }
  }, []);

  useEffect(() => {
    // Check immediately
    checkReady();

    // Poll every 2 seconds until ready
    const interval = setInterval(() => {
      checkReady();
    }, 2000);

    return () => clearInterval(interval);
  }, [checkReady]);

  useEffect(() => {
    if (status.ready) {
      setIsChecking(false);
    }
  }, [status.ready]);

  return { status, isChecking, checkReady };
}
