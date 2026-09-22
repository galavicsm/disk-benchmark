import {
  createContext,
  type Dispatch,
  type ReactNode,
  useContext,
  useEffect,
  useReducer,
} from "react";
import { EventsOn } from "../../../wailsjs/runtime/runtime";
import {
  type CancelledEvent,
  type CompletedEvent,
  type ExecutionAction,
  type ExecutionState,
  type FailedEvent,
  type ProgressEvent,
  type StateEvent,
  executionReducer,
  initialExecutionState,
} from "./state";

interface ExecutionContextValue {
  state: ExecutionState;
  dispatch: Dispatch<ExecutionAction>;
}

const ExecutionContext = createContext<ExecutionContextValue | null>(null);

export function ExecutionProvider({ children }: { children: ReactNode }) {
  const [state, dispatch] = useReducer(executionReducer, initialExecutionState);

  useEffect(() => {
    const unsubscribe = [
      EventsOn("benchmark:state", (event: StateEvent) => {
        dispatch({ type: "state-event", event });
      }),
      EventsOn("benchmark:progress", (event: ProgressEvent) => {
        dispatch({ type: "progress-event", event });
      }),
      EventsOn("benchmark:completed", (event: CompletedEvent) => {
        dispatch({ type: "completed-event", event });
      }),
      EventsOn("benchmark:failed", (event: FailedEvent) => {
        dispatch({ type: "failed-event", event });
      }),
      EventsOn("benchmark:cancelled", (event: CancelledEvent) => {
        dispatch({ type: "cancelled-event", event });
      }),
    ];
    return () => {
      for (const removeListener of unsubscribe) {
        removeListener();
      }
    };
  }, []);

  return (
    <ExecutionContext.Provider value={{ state, dispatch }}>
      {children}
    </ExecutionContext.Provider>
  );
}

export function useExecution(): ExecutionContextValue {
  const context = useContext(ExecutionContext);
  if (context === null) {
    throw new Error("useExecution must be used within an ExecutionProvider");
  }
  return context;
}
