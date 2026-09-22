import { useEffect, useState } from "react";
import type { app } from "../wailsjs/go/models";
import { GetDefaults } from "../wailsjs/go/app/Service";
import { BenchmarkForm } from "./features/configuration/BenchmarkForm";
import { ExecutionProvider } from "./features/execution/ExecutionContext";

type LoadState =
  | { status: "loading" }
  | { status: "ready"; defaults: app.BenchmarkRequest }
  | { status: "failed"; message: string };

function App() {
  const [loadState, setLoadState] = useState<LoadState>({ status: "loading" });

  useEffect(() => {
    let active = true;
    void GetDefaults()
      .then((defaults) => {
        if (active) {
          setLoadState({ status: "ready", defaults });
        }
      })
      .catch((error: unknown) => {
        if (active) {
          setLoadState({
            status: "failed",
            message: error instanceof Error ? error.message : String(error),
          });
        }
      });
    return () => {
      active = false;
    };
  }, []);

  return (
    <ExecutionProvider>
      <main className="app-shell">
      <header className="app-header">
        <div>
          <p className="eyebrow">Native disk performance utility</p>
          <h1>DiskBenchmark</h1>
        </div>
        <span className="desktop-badge">Desktop</span>
      </header>

      {loadState.status === "loading" && (
        <section className="loading-panel" aria-live="polite">
          Loading benchmark defaults...
        </section>
      )}
      {loadState.status === "failed" && (
        <section className="error-banner" role="alert">
          <strong>Application service unavailable</strong>
          <span>{loadState.message}</span>
        </section>
      )}
      {loadState.status === "ready" && (
        <BenchmarkForm initialRequest={loadState.defaults} />
      )}
      </main>
    </ExecutionProvider>
  );
}

export default App;
