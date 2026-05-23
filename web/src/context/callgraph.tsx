import { createContext, useContext, type ParentComponent } from 'solid-js';
import { createSignal, createEffect, on } from 'solid-js';
import {
  type CallNode, type CallEdge, type CallGraphStats, type ModelInfo,
  getCallGraphStats, getCallGraphNodes, getCallGraphEdges,
  buildCallGraph, getModels, getCallGraphModel, setCallGraphModel,
  getCallGraphBuildStatus,
} from '../api/client';
import { useServer } from './server';

interface CallGraphContextValue {
  stats: () => CallGraphStats | null;
  nodes: () => CallNode[];
  edges: () => CallEdge[];
  loading: () => boolean;
  building: () => boolean;
  buildSessionId: () => string | null;
  models: () => ModelInfo[];
  selectedModel: () => string;
  selectModel: (id: string) => void;
  selectedNode: () => CallNode | null;
  setSelectedNode: (node: CallNode | null) => void;
  refresh: () => Promise<void>;
  build: (rebuild?: boolean) => Promise<void>;
}

const CallGraphContext = createContext<CallGraphContextValue>();

export const CallGraphProvider: ParentComponent = (props) => {
  const server = useServer();
  const [stats, setStats] = createSignal<CallGraphStats | null>(null);
  const [nodes, setNodes] = createSignal<CallNode[]>([]);
  const [edges, setEdges] = createSignal<CallEdge[]>([]);
  const [loading, setLoading] = createSignal(false);
  const [building, setBuilding] = createSignal(false);
  const [buildSessionId, setBuildSessionId] = createSignal<string | null>(null);
  const [models, setModels] = createSignal<ModelInfo[]>([]);
  const [selectedModelId, setSelectedModelId] = createSignal<string>('');
  const [selectedNode, setSelectedNode] = createSignal<CallNode | null>(null);

  // Load models and persisted model preference whenever the directory changes
  createEffect(on(server.directory, (dir) => {
    getModels()
      .then((list) => setModels(list || []))
      .catch((e) => console.error('callgraph: load models failed:', e));

    getCallGraphModel(dir || undefined)
      .then((res) => { if (res.model) setSelectedModelId(res.model); })
      .catch((e) => console.error('callgraph: load model pref failed:', e));
  }));

  const selectedModel = (): string => {
    const id = selectedModelId();
    const enabled = models().filter((m) => m.enabled);
    if (id && enabled.some((m) => m.id === id)) return id;
    const def = enabled.find((m) => m.default) || enabled[0];
    return def?.id || '';
  };

  async function selectModel(id: string) {
    setSelectedModelId(id);
    try {
      await setCallGraphModel(id, server.directory() || undefined);
    } catch (e) {
      console.error('callgraph: save model pref failed:', e);
    }
  }

  async function refresh() {
    const dir = server.directory();
    if (!dir) return;
    setLoading(true);
    try {
      const [s, n, e] = await Promise.all([
        getCallGraphStats(dir),
        getCallGraphNodes(dir),
        getCallGraphEdges(dir),
      ]);
      setStats(s);
      setNodes(n || []);
      setEdges(e || []);
    } catch (e) {
      console.error('refresh call graph failed:', e);
    } finally {
      setLoading(false);
    }
  }

  async function build(rebuild = false) {
    if (building()) return;
    setBuilding(true);
    try {
      const res = await buildCallGraph(server.directory() || undefined, rebuild, selectedModel() || undefined);
      setBuildSessionId(res.sessionId);
    } catch (e) {
      console.error('start call graph build failed:', e);
      setBuilding(false);
    }
  }

  // Load when directory changes
  createEffect(on(server.directory, (dir) => {
    if (dir) refresh();
  }));

  // Refresh on SSE reconnect; defer: true skips the initial mount firing
  // (the directory effect already handles first load).
  // Always check build status on reconnect — handles both the missed-event case
  // and page-reload-during-build crash recovery.
  createEffect(on(server.connected, (isConnected) => {
    if (!isConnected) return;
    refresh();
    getCallGraphBuildStatus()
      .then((status) => {
        if (status.running) {
          setBuilding(true);
          setBuildSessionId(status.sessionId || null);
        } else if (buildSessionId()) {
          setBuilding(false);
          setBuildSessionId(null);
        }
      })
      .catch((e) => console.error('callgraph: build status check failed:', e));
  }, { defer: true }));

  // When the build agent finishes, refresh graph data and clear building state
  createEffect(on(server.eventTick, () => {
    const last = server.lastEvent();
    if (!last || last.type !== 'callgraph.built') return;
    const sid = (last.properties as any)?.sessionId;
    if (sid && sid === buildSessionId()) {
      setBuilding(false);
      setBuildSessionId(null);
      refresh();
    }
  }));

  const value: CallGraphContextValue = {
    stats,
    nodes,
    edges,
    loading,
    building,
    buildSessionId,
    models,
    selectedModel,
    selectModel,
    selectedNode,
    setSelectedNode,
    refresh,
    build,
  };

  return (
    <CallGraphContext.Provider value={value}>
      {props.children}
    </CallGraphContext.Provider>
  );
};

export function useCallGraph() {
  const ctx = useContext(CallGraphContext);
  if (!ctx) throw new Error('useCallGraph must be used within CallGraphProvider');
  return ctx;
}