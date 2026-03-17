import { useServerReady } from "../../hooks/useServerReady";
import { Loader2, CheckCircle2, AlertCircle } from "lucide-react";

export function LoadingOverlay() {
  const { status, isChecking } = useServerReady();

  // If ready, don't show anything
  if (status.ready) {
    return null;
  }

  return (
    <div className="fixed inset-0 bg-white/90 backdrop-blur-sm flex items-center justify-center z-50">
      <div className="bg-white border border-gray-200 rounded-lg shadow-lg p-8 max-w-md w-full mx-4">
        <div className="flex flex-col items-center text-center space-y-4">
          {/* Loading Icon */}
          <div className="relative">
            <Loader2 className="w-12 h-12 text-blue-500 animate-spin" />
          </div>

          {/* Title */}
          <h2 className="text-xl font-semibold text-gray-900">
            Loading Dictionary...
          </h2>

          {/* Description */}
          <p className="text-gray-600 text-sm">
            The spell checker is loading the dictionary in the background. This
            may take 15-30 seconds depending on the dictionary size.
          </p>

          {/* Status Badge */}
          <div className="inline-flex items-center gap-2 px-3 py-1.5 bg-blue-50 text-blue-700 rounded-full text-sm">
            <div className="w-2 h-2 bg-blue-500 rounded-full animate-pulse" />
            <span>{status.message || "Initializing..."}</span>
          </div>

          {/* Progress Info */}
          {status.loadTime && status.loadTime > 0 && (
            <p className="text-xs text-gray-500">
              Previous load took {status.loadTime.toFixed(1)}s
            </p>
          )}

          {/* Error State */}
          {!isChecking &&
            !status.ready &&
            status.message?.includes("unreachable") && (
              <div className="flex items-center gap-2 text-red-600 text-sm">
                <AlertCircle className="w-4 h-4" />
                <span>Cannot connect to server</span>
              </div>
            )}
        </div>
      </div>
    </div>
  );
}
