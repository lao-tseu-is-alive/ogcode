import { createContext, useContext, type ParentComponent } from 'solid-js';
import { createSignal } from 'solid-js';
import { getPath, getConfig, getVCS, getMode } from '../api/client';
import { createSSE, type SSEEvent } from '../api/sse';

interface ServerContextValue {
  directory: () => string;
  branch: () => string;
  mode: () => 'build' | 'plan';
  connected: () => boolean;
  memoryEnabled: () => boolean;
  memoryProvider: () => string;
  // Monotonically increasing counter that ticks on every relevant SSE event.
  // Consumers use this as a reactive dependency to know when to re-fetch.
  eventTick: () => number;
  lastEvent: () => SSEEvent | null;
}

const ServerContext = createContext<ServerContextValue>();

export const ServerProvider: ParentComponent = (props) => {
  const [directory, setDirectory] = createSignal('');
  const [branch, setBranch] = createSignal('');
  const [mode, setMode] = createSignal<'build' | 'plan'>('build');
  const [connected, setConnected] = createSignal(false);
  const [memoryEnabled, setMemoryEnabled] = createSignal(false);
  const [memoryProvider, setMemoryProvider] = createSignal('');
  const [eventTick, setEventTick] = createSignal(0);
  const [lastEvent, setLastEvent] = createSignal<SSEEvent | null>(null);

  // Load server info
  getPath().then((info) => {
    setDirectory(info.directory);
  }).catch(() => { /* ignore */ });

  // Load VCS branch info
  getVCS().then((info) => {
    if (info.branch) setBranch(info.branch);
  }).catch(() => { /* ignore */ });

  // Load server mode
  getMode().then((info) => {
    if (info.mode) setMode(info.mode as 'build' | 'plan');
  }).catch(() => { /* ignore */ });

  getConfig().then((config) => {
    setMemoryEnabled(config.memoryEnabled);
    setMemoryProvider(config.memoryProvider ?? '');
  }).catch(() => { /* ignore */ });

  // Connect to SSE
  createSSE('/event', (event) => {
    if (event.type === 'server.connected') {
      setConnected(true);
    } else if (event.type === 'server.config') {
      setMemoryEnabled(!!event.properties?.memoryEnabled);
      setMemoryProvider(event.properties?.memoryProvider ?? '');
    } else if (event.type === 'server.heartbeat') {
      // keep alive
    } else {
      setLastEvent(event);
      setEventTick((n) => n + 1);
    }
  });

  const value: ServerContextValue = {
    directory,
    branch,
    mode,
    connected,
    memoryEnabled,
    memoryProvider,
    eventTick,
    lastEvent,
  };

  return (
    <ServerContext.Provider value={value}>
      {props.children}
    </ServerContext.Provider>
  );
};

export function useServer() {
  const ctx = useContext(ServerContext);
  if (!ctx) throw new Error('useServer must be used within ServerProvider');
  return ctx;
}