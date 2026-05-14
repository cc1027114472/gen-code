import { useState } from "react";

declare global {
  interface Window {
    go?: {
      main?: {
        App?: {
          GetAppInfo: () => Promise<string>;
        };
      };
    };
  }
}

export default function App() {
  const [message, setMessage] = useState("Waiting for Go bridge...");
  const [error, setError] = useState("");

  const handleCheck = async () => {
    setError("");

    try {
      const result = await window.go?.main?.App?.GetAppInfo?.();
      setMessage(result ?? "Go bridge not available");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown bridge error");
    }
  };

  return (
    <main className="shell">
      <section className="panel">
        <p className="eyebrow">Wails Desktop Shell</p>
        <h1>Gen Code Desktop</h1>
        <p className="lead">
          The standalone desktop shell is running and ready for future business integration.
        </p>
        <button onClick={handleCheck}>Check Go Binding</button>
        <p className="message">{message}</p>
        {error ? <p className="error">{error}</p> : null}
      </section>
    </main>
  );
}
