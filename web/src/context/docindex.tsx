import { createContext, useContext, type ParentComponent } from 'solid-js';
import { createSignal, createEffect, on } from 'solid-js';
import {
  type DocSummary, type ModelInfo,
  getDocIndexBuildStatus, buildDocIndex, getIndexedDocs, getModels,
} from '../api/client';
import { useServer } from './server';

interface DocIndexContextValue {
  docs: () => DocSummary[];
  loading: () => boolean;
  building: () => boolean;
  models: () => ModelInfo[];
  selectedModel: () => string;
  selectModel: (id: string) => void;
  refresh: () => Promise<void>;
  build: (rebuild?: boolean) => Promise<void>;
}

const DocIndexContext = createContext<DocIndexContextValue>();

export const DocIndexProvider: ParentComponent = (props) => {
  const server = useServer();
  const [docs, setDocs] = createSignal<DocSummary[]>([]);
  const [loading, setLoading] = createSignal(false);
  const [building, setBuilding] = createSignal(false);
  const [models, setModels] = createSignal<ModelInfo[]>([]);
  const [selectedModelId, setSelectedModelId] = createSignal<string>('');

  // Load models whenever the directory changes
  createEffect(on(server.directory, () => {
    getModels()
      .then((list) => setModels(list || []))
      .catch((e) => console.error('docindex: load models failed:', e));
  }));

  const selectedModel = (): string => {
    const id = selectedModelId();
    const enabled = models().filter((m) => m.enabled);
    if (id && enabled.some((m) => m.id === id)) return id;
    const def = enabled.find((m) => m.default) || enabled[0];
    return def?.id || '';
  };

  const selectModel = (id: string) => setSelectedModelId(id);

  async function refresh() {
    const dir = server.directory();
    if (!dir) return;
    setLoading(true);
    try {
      const d = await getIndexedDocs(dir);
      setDocs(d || []);
    } catch (e) {
      console.error('docindex refresh failed:', e);
    } finally {
      setLoading(false);
    }
  }

  async function build(rebuild = false) {
    if (building()) return;
    setBuilding(true);
    try {
      await buildDocIndex(server.directory() || undefined, rebuild, selectedModel() || undefined);
    } catch (e) {
      console.error('start docindex build failed:', e);
      setBuilding(false);
    }
  }

  // Load when directory changes
  createEffect(on(server.directory, (dir) => {
    if (dir) refresh();
  }));

  // Refresh on SSE reconnect; always check build status on reconnect for crash recovery.
  createEffect(on(server.connected, (isConnected) => {
    if (!isConnected) return;
    refresh();
    getDocIndexBuildStatus()
      .then((status) => {
        if (status.running) {
          setBuilding(true);
        } else if (building()) {
          setBuilding(false);
        }
      })
      .catch((e) => console.error('docindex: build status check failed:', e));
  }, { defer: true }));

  // When the build finishes, refresh doc list and clear building state
  createEffect(on(server.eventTick, () => {
    const last = server.lastEvent();
    if (!last || last.type !== 'docindex.built') return;
    setBuilding(false);
    refresh();
  }));

  const value: DocIndexContextValue = {
    docs,
    loading,
    building,
    models,
    selectedModel,
    selectModel,
    refresh,
    build,
  };

  return (
    <DocIndexContext.Provider value={value}>
      {props.children}
    </DocIndexContext.Provider>
  );
};

export function useDocIndex() {
  const ctx = useContext(DocIndexContext);
  if (!ctx) throw new Error('useDocIndex must be used within DocIndexProvider');
  return ctx;
}
