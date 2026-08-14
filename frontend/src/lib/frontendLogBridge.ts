import * as PlatformService from "../../bindings/account-switcher/internal/platform/platformservice.js";

/**
 * Forwards what the webview prints into the application log.
 *
 * The console is invisible to that log otherwise, which has twice meant a fault
 * could only be explained by asking the user to open devtools and read it back.
 *
 * Errors, uncaught exceptions and rejected promises are always forwarded: they
 * are rare, and they are exactly what goes missing. Everything chattier is only
 * forwarded while debug logging is on, so an ordinary session does not write a
 * line per render. The Go side applies the same rule, so neither end has to
 * trust the other to have got it right.
 */
export function installFrontendLogBridge(): () => void {
  const original = {
    log: console.log,
    info: console.info,
    warn: console.warn,
    error: console.error,
    debug: console.debug,
  };

  const send = (level: string, args: unknown[]): void => {
    let text: string;
    try {
      text = args
        .map((a) => {
          if (typeof a === "string") return a;
          if (a instanceof Error) return `${a.name}: ${a.message}\n${a.stack ?? ""}`;
          try {
            return JSON.stringify(a);
          } catch {
            return String(a);
          }
        })
        .join(" ");
    } catch {
      return;
    }
    // Bounded: a stack trace is useful, a megabyte of state is not.
    if (text.length > 4000) text = text.slice(0, 4000) + "...";
    // Never awaited, and failures are swallowed: logging must not be able to
    // break the thing it is reporting on.
    void PlatformService.LogFrontend(level, text).catch(() => {});
  };

  const wrap = (name: keyof typeof original, level: string) => {
    console[name] = ((...args: unknown[]) => {
      original[name].apply(console, args as []);
      send(level, args);
    }) as typeof console.log;
  };

  wrap("log", "info");
  wrap("info", "info");
  wrap("debug", "debug");
  wrap("warn", "warn");
  wrap("error", "error");

  const onError = (ev: ErrorEvent): void => {
    send("error", [`uncaught: ${ev.message}`, `${ev.filename}:${ev.lineno}:${ev.colno}`, ev.error]);
  };
  const onRejection = (ev: PromiseRejectionEvent): void => {
    send("error", ["unhandled rejection:", ev.reason]);
  };
  window.addEventListener("error", onError);
  window.addEventListener("unhandledrejection", onRejection);

  return () => {
    console.log = original.log;
    console.info = original.info;
    console.warn = original.warn;
    console.error = original.error;
    console.debug = original.debug;
    window.removeEventListener("error", onError);
    window.removeEventListener("unhandledrejection", onRejection);
  };
}
