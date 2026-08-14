import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./app";
import "./styles.css";

const root = document.getElementById("root");
if (root === null) {
  throw new TypeError("Missing application root element");
}

if (import.meta.env.DEV) {
  void import("react-scan").then(({ scan }) => scan({ enabled: true }));
}

createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
