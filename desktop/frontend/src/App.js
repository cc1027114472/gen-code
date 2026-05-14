import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState } from "react";
export default function App() {
    const [message, setMessage] = useState("Waiting for Go bridge...");
    const [error, setError] = useState("");
    const handleCheck = async () => {
        setError("");
        try {
            const result = await window.go?.main?.App?.GetAppInfo?.();
            setMessage(result ?? "Go bridge not available");
        }
        catch (err) {
            setError(err instanceof Error ? err.message : "Unknown bridge error");
        }
    };
    return (_jsx("main", { className: "shell", children: _jsxs("section", { className: "panel", children: [_jsx("p", { className: "eyebrow", children: "Wails Desktop Shell" }), _jsx("h1", { children: "Gen Code Desktop" }), _jsx("p", { className: "lead", children: "The standalone desktop shell is running and ready for future business integration." }), _jsx("button", { onClick: handleCheck, children: "Check Go Binding" }), _jsx("p", { className: "message", children: message }), error ? _jsx("p", { className: "error", children: error }) : null] }) }));
}
