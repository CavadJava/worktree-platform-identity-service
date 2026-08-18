import { useEffect } from 'react';
import { message } from 'antd';

export function useQueryErrorToast(isError: boolean, error: unknown) {
  useEffect(() => {
    if (isError) {
      message.error(error instanceof Error ? error.message : 'Xəta baş verdi');
    }
  }, [isError, error]);
}
