import { createSignal, createEffect, onMount, onCleanup, Show, For } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import { useServer } from '../context/server';
import { useCallGraph } from '../context/callgraph';
import type { CallNode, CallNodeDetail, CallNodeSummary } from '../api/client';
import { getCallGraphNodeDetail, searchCallGraph } from '../api/client';
import SessionSidebar from '../components/session-sidebar';
import PlanSidebar from '../components/plan-sidebar';

// ─── D3 force graph ───

interface SimNode extends d3.SimulationNodeDatum {
  id: number;
  symbol: string;
  pkg: string;
  kind: string;
  filePath: string;
  line: number;
  doc: string;
}

interface SimLink extends d3.SimulationLinkDatum<SimNode> {
  source: number | SimNode;
  target: number | SimNode;
  callType: string;
}

import * as d3 from 'd3';

function Sidebar() {
  const server = useServer();
  return (
    <Show when={server.mode() === 'plan'} fallback={<SessionSidebar />}>
      <PlanSidebar />
    </Show>
  );
}

// Color palette by package (hashed)
const PACKAGE_COLORS = [
  '#3b82f6', '#8b5cf6', '#ec4899', '#f97316', '#14b8a6',
  '#eab308', '#06b6d4', '#84cc16', '#f43f5e', '#6366f1',
  '#22d3ee', '#a3e635', '#fb923c', '#c084fc', '#2dd4bf',
];

function pkgColor(pkg: string): string {
  let hash = 0;
  for (let i = 0; i < pkg.length; i++) {
    hash = ((hash << 5) - hash + pkg.charCodeAt(i)) | 0;
  }
  return PACKAGE_COLORS[Math.abs(hash) % PACKAGE_COLORS.length];
}

function kindIcon(kind: string): string {
  switch (kind) {
    // Callable
    case 'function':    return 'F';
    case 'method':      return 'M';
    case 'constructor': return 'C';
    case 'init':        return 'I';
    // Structural
    case 'type':        return 'T';
    case 'interface':   return 'If';
    case 'enum':        return 'E';
    // Values
    case 'const':       return 'K';
    case 'variable':    return 'V';
    // Organization
    case 'module':      return 'Mod';
    case 'macro':       return 'Mc';
    default:            return '?';
  }
}

function kindRadius(kind: string): number {
  switch (kind) {
    case 'module':      return 12;
    case 'type':        return 10;
    case 'interface':   return 9;
    case 'enum':        return 8;
    case 'method':      return 7;
    case 'constructor': return 7;
    case 'function':    return 6;
    case 'const':       return 5;
    case 'variable':    return 5;
    case 'macro':       return 5;
    case 'init':        return 5;
    default:            return 6;
  }
}

function truncateLabel(s: string, max = 22): string {
  return s.length > max ? s.slice(0, max - 1) + '…' : s;
}

// ─── Detail panel ───

function NodeDetailPanel(props: { detail: CallNodeDetail | null; loading: boolean; onSelectNode: (id: number) => void }) {
  const navigate = useNavigate();

  return (
    <Show when={props.detail !== null} fallback={
      <div class="flex flex-col items-center justify-center h-full text-center px-6 py-12">
        <svg class="w-12 h-12 text-zinc-700 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M13.81 9.06l.28.28a2.69 2.69 0 010 3.81l-5.66 5.66a2.69 2.69 0 01-3.81 0l-.28-.28a2.69 2.69 0 010-3.81l5.66-5.66a2.69 2.69 0 013.81 0zM17.19 9.06l.28.28a2.69 2.69 0 010 3.81l-.28.28" />
        </svg>
        <p class="text-[13px] text-zinc-500 font-medium">Call Graph Explorer</p>
        <p class="text-[12px] text-zinc-600 mt-1.5 max-w-[200px]">
          Click any node in the graph to see its details, callers, and callees.
        </p>
      </div>
    }>
      {() => {
        const d = props.detail!;
        return (
          <div class="flex flex-col h-full overflow-hidden">
            {/* Header */}
            <div class="shrink-0 px-4 pt-4 pb-3 border-b border-[color:var(--border-subtle)]">
              <div class="flex items-center gap-2">
                <span
                  class="inline-flex items-center justify-center w-6 h-6 rounded text-[10px] font-bold shrink-0"
                  style={{ background: pkgColor(d.node.package) + '22', color: pkgColor(d.node.package) }}
                >
                  {kindIcon(d.node.kind)}
                </span>
                <div class="min-w-0 flex-1">
                  <h3 class="text-[14px] font-semibold text-zinc-100 truncate">{d.node.symbol}</h3>
                  <p class="text-[11px] text-zinc-500 font-mono truncate">{d.node.package}</p>
                </div>
              </div>
              <div class="flex items-center gap-2 mt-2 flex-wrap">
                <span class="text-[10px] px-1.5 py-0.5 rounded bg-[color:var(--bg-elevated)] border border-[color:var(--border-subtle)] text-zinc-400 font-mono">
                  {d.node.kind}
                </span>
                <span class="text-[10px] text-zinc-500 font-mono">
                  {d.node.filePath}:{d.node.line}
                </span>
              </div>
              <Show when={d.node.signature}>
                <div class="mt-2 px-2 py-1.5 rounded bg-[color:var(--bg-base)] border border-[color:var(--border-subtle)] text-[11px] font-mono text-zinc-400 break-all">
                  {d.node.signature}
                </div>
              </Show>
            </div>

            {/* Doc */}
            <Show when={d.node.doc}>
              <div class="shrink-0 px-4 py-3 border-b border-[color:var(--border-subtle)]">
                <p class="text-[11px] text-zinc-400 leading-relaxed">{d.node.doc}</p>
              </div>
            </Show>

            {/* Scrollable callees + callers */}
            <div class="flex-1 overflow-y-auto">
              {/* Callees */}
              <div class="px-4 py-3">
                <h4 class="text-[11px] font-semibold text-zinc-500 uppercase tracking-wider mb-2">
                  Calls ({d.callees.length})
                </h4>
                <Show when={d.callees.length === 0} fallback={
                  <div class="space-y-1">
                    <For each={d.callees}>
                      {(c) => (
                        <button
                          onClick={() => props.onSelectNode(c.id)}
                          class="w-full text-left flex items-center gap-2 px-2 py-1.5 rounded-md hover:bg-[color:var(--bg-hover)]/50 transition group"
                        >
                          <span
                            class="w-2 h-2 rounded-full shrink-0"
                            style={{ background: pkgColor(c.package) }}
                          />
                          <span class="text-[12px] text-zinc-300 group-hover:text-zinc-100 truncate flex-1 min-w-0 font-mono">
                            {truncateLabel(c.symbol)}
                          </span>
                          <span class="text-[10px] text-zinc-600 shrink-0">{c.callType}</span>
                        </button>
                      )}
                    </For>
                  </div>
                }>
                  <p class="text-[11px] text-zinc-600 italic">No outgoing calls</p>
                </Show>
              </div>

              {/* Callers */}
              <div class="px-4 py-3">
                <h4 class="text-[11px] font-semibold text-zinc-500 uppercase tracking-wider mb-2">
                  Called by ({d.callers.length})
                </h4>
                <Show when={d.callers.length === 0} fallback={
                  <div class="space-y-1">
                    <For each={d.callers}>
                      {(c) => (
                        <button
                          onClick={() => props.onSelectNode(c.id)}
                          class="w-full text-left flex items-center gap-2 px-2 py-1.5 rounded-md hover:bg-[color:var(--bg-hover)]/50 transition group"
                        >
                          <span
                            class="w-2 h-2 rounded-full shrink-0"
                            style={{ background: pkgColor(c.package) }}
                          />
                          <span class="text-[12px] text-zinc-300 group-hover:text-zinc-100 truncate flex-1 min-w-0 font-mono">
                            {truncateLabel(c.symbol)}
                          </span>
                          <span class="text-[10px] text-zinc-600 shrink-0">{c.callType}</span>
                        </button>
                      )}
                    </For>
                  </div>
                }>
                  <p class="text-[11px] text-zinc-600 italic">No incoming calls</p>
                </Show>
              </div>
            </div>
          </div>
        );
      }}
    </Show>
  );
}

// ─── Main page ───

export default function CallGraphExplorer() {
  const server = useServer();
  const cg = useCallGraph();
  const navigate = useNavigate();
  const [searchQuery, setSearchQuery] = createSignal('');
  const [searchResults, setSearchResults] = createSignal<CallNode[]>([]);
  const [searching, setSearching] = createSignal(false);
  const [showSearch, setShowSearch] = createSignal(false);
  const [selectedDetail, setSelectedDetail] = createSignal<CallNodeDetail | null>(null);
  const [detailLoading, setDetailLoading] = createSignal(false);
  const [packageFilter, setPackageFilter] = createSignal('');
  const [kindFilter, setKindFilter] = createSignal('');

  const [showBuildModal, setShowBuildModal] = createSignal(false);
  const [modalIsRebuild, setModalIsRebuild] = createSignal(false);

  const openBuildModal = (rebuild: boolean) => {
    setModalIsRebuild(rebuild);
    setShowBuildModal(true);
  };

  const handleConfirmBuild = () => {
    setShowBuildModal(false);
    cg.build(modalIsRebuild());
  };

  let svgRef: SVGSVGElement | undefined;
  let containerRef: HTMLDivElement | undefined;

  const handleSelectNode = async (nodeID: number) => {
    setDetailLoading(true);
    try {
      const detail = await getCallGraphNodeDetail(nodeID);
      setSelectedDetail(detail);
    } catch (e) {
      console.error('failed to load node detail:', e);
    } finally {
      setDetailLoading(false);
    }
  };

  const handleSearch = async () => {
    const q = searchQuery().trim();
    if (!q) return;
    setSearching(true);
    try {
      const results = await searchCallGraph(q, server.directory());
      setSearchResults(results || []);
    } catch (e) {
      console.error('call graph search failed:', e);
    } finally {
      setSearching(false);
    }
  };

  // Unique packages for filter dropdown
  const packages = () => {
    const pkgs = new Set(cg.nodes().map(n => n.package));
    return Array.from(pkgs).sort();
  };

  // Filtered nodes for force graph
  const filteredNodes = () => {
    let nodes = cg.nodes();
    const pf = packageFilter();
    const kf = kindFilter();
    if (pf) nodes = nodes.filter(n => n.package === pf);
    if (kf) nodes = nodes.filter(n => n.kind === kf);
    return nodes;
  };

  const filteredEdges = () => {
    const nodeIds = new Set(filteredNodes().map(n => n.id));
    return cg.edges().filter(e => nodeIds.has(e.callerId) && nodeIds.has(e.calleeId));
  };

  // D3 force simulation
  let simulation: d3.Simulation<SimNode, SimLink> | undefined;
  let zoomBehavior: d3.ZoomBehavior<SVGSVGElement, unknown> | undefined;

  const renderGraph = () => {
    if (!svgRef || !containerRef) return;

    const nodes = filteredNodes();
    const edges = filteredEdges();
    if (nodes.length === 0) return;

    // Clear previous
    if (simulation) {
      simulation.stop();
      simulation = undefined;
    }

    const svg = d3.select(svgRef);
    svg.selectAll('*').remove();

    const width = containerRef.clientWidth;
    const height = containerRef.clientHeight;

    svg.attr('width', width).attr('height', height);

    const g = svg.append('g');

    // Zoom
    zoomBehavior = d3.zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.1, 4])
      .on('zoom', (event: d3.D3ZoomEvent<SVGSVGElement, unknown>) => {
        g.attr('transform', event.transform.toString());
      });

    svg.call(zoomBehavior);

    // Build simulation data
    const simNodes: SimNode[] = nodes.map(n => ({
      id: n.id,
      symbol: n.symbol,
      pkg: n.package,
      kind: n.kind,
      filePath: n.filePath,
      line: n.line,
      doc: n.doc || '',
    }));

    const nodeIdMap = new Map(simNodes.map(n => [n.id, n]));
    // O(1) lookup for the original CallNode — used in the click handler to set
    // cg.selectedNode() so mouseleave can tell a node is still selected.
    const callNodeMap = new Map(nodes.map(n => [n.id, n]));

    const simLinks: SimLink[] = edges
      .filter(e => nodeIdMap.has(e.callerId) && nodeIdMap.has(e.calleeId))
      .map(e => ({
        source: e.callerId,
        target: e.calleeId,
        callType: e.callType,
      }));

    // Arrow marker definitions
    const defs = svg.append('defs');

    const callTypes = new Set(edges.map(e => e.callType));
    const linkColors: Record<string, string> = {
      direct: '#52525b',
      dynamic: '#f97316',
      interface: '#8b5cf6',
      callback: '#14b8a6',
    };

    for (const ct of callTypes) {
      defs.append('marker')
        .attr('id', `arrow-${ct}`)
        .attr('viewBox', '0 -5 10 10')
        .attr('refX', 20)
        .attr('refY', 0)
        .attr('markerWidth', 6)
        .attr('markerHeight', 6)
        .attr('orient', 'auto')
        .append('path')
        .attr('d', 'M0,-5L10,0L0,5')
        .attr('fill', linkColors[ct] || '#52525b');
    }

    // Links
    const link = g.append('g')
      .selectAll('line')
      .data(simLinks)
      .join('line')
      .attr('stroke', (d: SimLink) => linkColors[d.callType] || '#52525b')
      .attr('stroke-width', 1.2)
      .attr('stroke-opacity', 0.5)
      .attr('marker-end', (d: SimLink) => `url(#arrow-${d.callType})`);

    // Node groups
    const nodeGroup = g.append('g')
      .selectAll<SVGGElement, SimNode>('g')
      .data(simNodes, (d: SimNode) => `${d.id}`)
      .join('g')
      .attr('cursor', 'pointer')
      .call(d3.drag<SVGGElement, SimNode, SimNode>()
        .on('start', (event: d3.D3DragEvent<SVGGElement, SimNode, SimNode>) => {
          if (!event.active) simulation?.alphaTarget(0.3).restart();
          event.subject.fx = event.subject.x;
          event.subject.fy = event.subject.y;
        })
        .on('drag', (event: d3.D3DragEvent<SVGGElement, SimNode, SimNode>) => {
          event.subject.fx = event.x;
          event.subject.fy = event.y;
        })
        .on('end', (event: d3.D3DragEvent<SVGGElement, SimNode, SimNode>) => {
          if (!event.active) simulation?.alphaTarget(0);
          event.subject.fx = null;
          event.subject.fy = null;
        })
      );

    // Node circles
    nodeGroup.append('circle')
      .attr('r', (d: SimNode) => kindRadius(d.kind))
      .attr('fill', (d: SimNode) => pkgColor(d.pkg))
      .attr('fill-opacity', 0.85)
      .attr('stroke', (d: SimNode) => pkgColor(d.pkg))
      .attr('stroke-width', 1.5)
      .attr('stroke-opacity', 0.3);

    // Node labels
    nodeGroup.append('text')
      .attr('dx', 10)
      .attr('dy', 3)
      .text((d: SimNode) => truncateLabel(d.symbol))
      .attr('fill', '#a1a1aa')
      .attr('font-size', '10px')
      .attr('font-family', 'var(--font-mono)')
      .attr('pointer-events', 'none');

    // Click handler
    nodeGroup.on('click', (_event: MouseEvent, d: SimNode) => {
      // Highlight on click
      nodeGroup.selectAll('circle')
        .attr('fill-opacity', 0.4)
        .attr('stroke-opacity', 0.15);
      d3.select(_event.currentTarget as SVGGElement).select('circle')
        .attr('fill-opacity', 1)
        .attr('stroke-opacity', 0.8)
        .attr('stroke-width', 2.5);

      // Highlight connected links
      link.attr('stroke-opacity', 0.08)
        .attr('stroke-width', 1);
      link.filter((l: SimLink) => {
        const s = l.source as SimNode | number;
        const t = l.target as SimNode | number;
        const srcId = typeof s === 'number' ? s : s.id;
        const tgtId = typeof t === 'number' ? t : t.id;
        return srcId === d.id || tgtId === d.id;
      })
        .attr('stroke-opacity', 0.7)
        .attr('stroke-width', 2);

      // Track selected node so mouseleave can preserve the highlight
      const found = callNodeMap.get(d.id);
      if (found) cg.setSelectedNode(found);

      handleSelectNode(d.id);
    });

    // Hover: highlight connected nodes
    nodeGroup.on('mouseenter', (_event: MouseEvent, d: SimNode) => {
      const connectedIds = new Set<number>();
      connectedIds.add(d.id);
      link.each((l: SimLink) => {
        const s = l.source as SimNode | number;
        const t = l.target as SimNode | number;
        const srcId = typeof s === 'number' ? s : s.id;
        const tgtId = typeof t === 'number' ? t : t.id;
        if (srcId === d.id) connectedIds.add(tgtId);
        if (tgtId === d.id) connectedIds.add(srcId);
      });

      nodeGroup.select('circle')
        .attr('fill-opacity', (nd: SimNode) => connectedIds.has(nd.id) ? 1 : 0.2)
        .attr('stroke-opacity', (nd: SimNode) => connectedIds.has(nd.id) ? 0.6 : 0.08);
      nodeGroup.select('text')
        .attr('fill', (nd: SimNode) => connectedIds.has(nd.id) ? '#e4e4e7' : '#3f3f46');
      link.attr('stroke-opacity', (l: SimLink) => {
        const s = l.source as SimNode | number;
        const t = l.target as SimNode | number;
        const srcId = typeof s === 'number' ? s : s.id;
        const tgtId = typeof t === 'number' ? t : t.id;
        return (srcId === d.id || tgtId === d.id) ? 0.7 : 0.06;
      });
    });

    nodeGroup.on('mouseleave', () => {
      // Only reset if no node is "selected"
      const selected = cg.selectedNode();
      if (!selected) {
        nodeGroup.select('circle').attr('fill-opacity', 0.85).attr('stroke-opacity', 0.3);
        nodeGroup.select('text').attr('fill', '#a1a1aa');
        link.attr('stroke-opacity', 0.5).attr('stroke-width', 1.2);
      }
    });

    // Force simulation
    simulation = d3.forceSimulation<SimNode>(simNodes)
      .force('link', d3.forceLink<SimNode, SimLink>(simLinks)
        .id((d: SimNode) => d.id)
        .distance(80)
      )
      .force('charge', d3.forceManyBody().strength(-200))
      .force('center', d3.forceCenter(width / 2, height / 2))
      .force('collision', d3.forceCollide().radius(12))
      .on('tick', () => {
        link
          .attr('x1', (d: SimLink) => (d.source as SimNode).x ?? 0)
          .attr('y1', (d: SimLink) => (d.source as SimNode).y ?? 0)
          .attr('x2', (d: SimLink) => (d.target as SimNode).x ?? 0)
          .attr('y2', (d: SimLink) => (d.target as SimNode).y ?? 0);

        nodeGroup.attr('transform', (d: SimNode) => `translate(${d.x ?? 0},${d.y ?? 0})`);
      });

    // Initial zoom to fit
    const padding = 60;
    const svgEl = svgRef!;
    const initialTransform = d3.zoomIdentity
      .translate(width / 2, height / 2)
      .scale(0.9)
      .translate(-width / 2, -height / 2);
    svg.call(zoomBehavior.transform, initialTransform);
  };

  // Re-render graph when data changes
  createEffect(() => {
    // Track dependencies
    const nodeCount = filteredNodes().length;
    const edgeCount = filteredEdges().length;
    if (nodeCount > 0) {
      // Use requestAnimationFrame to ensure DOM is ready
      requestAnimationFrame(() => renderGraph());
    }
  });

  onCleanup(() => {
    if (simulation) {
      simulation.stop();
    }
  });

  // Handle window resize
  const handleResize = () => {
    if (simulation) {
      simulation.stop();
    }
    renderGraph();
  };

  onMount(() => {
    window.addEventListener('resize', handleResize);
  });

  onCleanup(() => {
    window.removeEventListener('resize', handleResize);
  });

  const handleSearchSelect = (node: CallNode) => {
    cg.setSelectedNode(node);
    handleSelectNode(node.id);
    setShowSearch(false);
  };

  const handleResetView = () => {
    cg.setSelectedNode(null);
    setSelectedDetail(null);
    renderGraph();
  };

  return (
    <div class="flex h-screen w-full">
      <Sidebar />

      <div class="flex-1 flex flex-col overflow-hidden relative bg-[color:var(--bg-base)]">
        {/* Header */}
        <div class="shrink-0 border-b border-[color:var(--border-subtle)] bg-[color:var(--bg-surface)] px-4 py-2.5 flex items-center gap-3">
          {/* Title */}
          <div class="flex items-center gap-2 shrink-0">
            <svg class="w-4 h-4 text-[color:var(--accent)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M13.81 9.06l.28.28a2.69 2.69 0 010 3.81l-5.66 5.66a2.69 2.69 0 01-3.81 0l-.28-.28a2.69 2.69 0 010-3.81l5.66-5.66a2.69 2.69 0 013.81 0zM17.19 9.06l.28.28a2.69 2.69 0 010 3.81l-.28.28" />
            </svg>
            <h1 class="text-[14px] font-semibold text-zinc-100">Call Graph</h1>
          </div>

          <Show when={cg.stats()}>
            <span class="text-[11px] text-zinc-500 bg-[color:var(--bg-elevated)] px-2 py-0.5 rounded-md border border-[color:var(--border-subtle)]">
              {cg.stats()!.nodes} nodes · {cg.stats()!.edges} edges
            </span>
          </Show>

          <div class="flex-1" />

          {/* Search */}
          <div class="relative">
            <button
              onClick={() => setShowSearch(!showSearch())}
              class={`h-7 px-2.5 rounded-md text-[12px] flex items-center gap-1.5 transition
                ${showSearch() ? 'bg-[color:var(--accent-soft)] text-[color:var(--accent)] border border-[color:var(--accent)]/30' : 'bg-[color:var(--bg-elevated)] text-zinc-400 border border-[color:var(--border-subtle)] hover:text-zinc-200 hover:border-[color:var(--border-default)]'}`}
            >
              <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-4.35-4.35M17 10a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
              Search
            </button>

            <Show when={showSearch()}>
              <div class="absolute right-0 top-9 w-72 bg-[color:var(--bg-surface)] border border-[color:var(--border-default)] rounded-lg shadow-lg z-50 overflow-hidden">
                <div class="p-2">
                  <div class="relative">
                    <svg class="w-3.5 h-3.5 text-zinc-500 absolute left-2.5 top-1/2 -translate-y-1/2 pointer-events-none" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-4.35-4.35M17 10a7 7 0 11-14 0 7 7 0 0114 0z" />
                    </svg>
                    <input
                      type="text"
                      value={searchQuery()}
                      onInput={(e) => setSearchQuery(e.currentTarget.value)}
                      onKeyDown={(e) => { if (e.key === 'Enter') handleSearch(); }}
                      placeholder="Search functions…"
                      class="w-full h-8 pl-8 pr-2 bg-[color:var(--bg-base)] border border-[color:var(--border-subtle)] rounded-md text-[12px] text-zinc-200 placeholder-zinc-600 focus:outline-none focus:border-[color:var(--border-default)] transition"
                      autofocus
                    />
                  </div>
                </div>

                <Show when={searching()}>
                  <div class="px-3 pb-2 flex items-center gap-2 text-[11px] text-zinc-500">
                    <div class="w-3 h-3 border border-[color:var(--accent)] border-t-transparent rounded-full animate-spin" />
                    Searching…
                  </div>
                </Show>

                <Show when={!searching() && searchResults().length > 0}>
                  <div class="max-h-64 overflow-y-auto border-t border-[color:var(--border-subtle)]">
                    <For each={searchResults()}>
                      {(n) => (
                        <button
                          onClick={() => handleSearchSelect(n)}
                          class="w-full text-left px-3 py-2 hover:bg-[color:var(--bg-hover)]/50 transition flex items-center gap-2"
                        >
                          <span class="w-2 h-2 rounded-full shrink-0" style={{ background: pkgColor(n.package) }} />
                          <div class="min-w-0 flex-1">
                            <div class="text-[12px] text-zinc-200 font-mono truncate">{n.symbol}</div>
                            <div class="text-[10px] text-zinc-500 truncate">{n.package} · {n.filePath}:{n.line}</div>
                          </div>
                          <span class="text-[10px] text-zinc-600">{n.kind}</span>
                        </button>
                      )}
                    </For>
                  </div>
                </Show>

                <Show when={!searching() && searchQuery() && searchResults().length === 0}>
                  <div class="px-3 py-4 text-center text-[11px] text-zinc-600">
                    No results found
                  </div>
                </Show>
              </div>
            </Show>
          </div>

          {/* Package filter */}
          <select
            value={packageFilter()}
            onChange={(e) => setPackageFilter(e.currentTarget.value)}
            class="h-7 px-2 rounded-md text-[12px] bg-[color:var(--bg-elevated)] border border-[color:var(--border-subtle)] text-zinc-400 hover:text-zinc-200 hover:border-[color:var(--border-default)] focus:outline-none focus:border-[color:var(--border-default)] transition cursor-pointer appearance-none pr-6"
            style={{ 'background-image': `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' fill='%2371717a' viewBox='0 0 24 24'%3E%3Cpath d='M7 10l5 5 5-5z'/%3E%3C/svg%3E")`, 'background-repeat': 'no-repeat', 'background-position': 'right 4px center' }}
          >
            <option value="">All packages</option>
            <For each={packages()}>
              {(p) => <option value={p}>{p}</option>}
            </For>
          </select>

          {/* Kind filter */}
          <select
            value={kindFilter()}
            onChange={(e) => setKindFilter(e.currentTarget.value)}
            class="h-7 px-2 rounded-md text-[12px] bg-[color:var(--bg-elevated)] border border-[color:var(--border-subtle)] text-zinc-400 hover:text-zinc-200 hover:border-[color:var(--border-default)] focus:outline-none focus:border-[color:var(--border-default)] transition cursor-pointer appearance-none pr-6"
            style={{ 'background-image': `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' fill='%2371717a' viewBox='0 0 24 24'%3E%3Cpath d='M7 10l5 5 5-5z'/%3E%3C/svg%3E")`, 'background-repeat': 'no-repeat', 'background-position': 'right 4px center' }}
          >
            <option value="">All kinds</option>
            <option value="function">Function</option>
            <option value="method">Method</option>
            <option value="constructor">Constructor</option>
            <option value="init">Init</option>
            <option value="type">Type</option>
            <option value="interface">Interface</option>
            <option value="enum">Enum</option>
            <option value="const">Const</option>
            <option value="variable">Variable</option>
            <option value="module">Module</option>
            <option value="macro">Macro</option>
          </select>

          {/* Rebuild (shown when graph has data) */}
          <Show when={(cg.stats()?.nodes ?? 0) > 0}>
            <button
              onClick={() => openBuildModal(true)}
              disabled={cg.building() || cg.loading()}
              class="h-7 px-2.5 rounded-md text-[12px] bg-[color:var(--bg-elevated)] border border-[color:var(--border-subtle)] text-zinc-400 hover:text-zinc-200 hover:border-[color:var(--border-default)] disabled:opacity-50 transition flex items-center gap-1.5"
              title="Rebuild call graph from scratch"
            >
              <Show when={cg.building()} fallback={
                <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                </svg>
              }>
                <div class="w-3 h-3 border-2 border-[color:var(--accent)] border-t-transparent rounded-full animate-spin" />
              </Show>
              Rebuild
            </button>
          </Show>

          {/* Refresh */}
          <button
            onClick={() => cg.refresh()}
            disabled={cg.loading() || cg.building()}
            class="h-7 px-2.5 rounded-md text-[12px] bg-[color:var(--bg-elevated)] border border-[color:var(--border-subtle)] text-zinc-400 hover:text-zinc-200 hover:border-[color:var(--border-default)] disabled:opacity-50 transition flex items-center gap-1.5"
          >
            <Show when={cg.loading()} fallback={
              <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
            }>
              <div class="w-3 h-3 border-2 border-[color:var(--accent)] border-t-transparent rounded-full animate-spin" />
            </Show>
            Refresh
          </button>
        </div>

        {/* Main content: graph + detail panel */}
        <div class="flex-1 flex overflow-hidden">
          {/* Graph area */}
          <div class="flex-1 relative" ref={containerRef}>
            <Show when={filteredNodes().length === 0 && !cg.loading()}>
              <div class="absolute inset-0 flex flex-col items-center justify-center text-center z-10 px-6">
                <Show when={cg.building()} fallback={
                  <>
                    <svg class="w-12 h-12 text-zinc-700 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M13.81 9.06l.28.28a2.69 2.69 0 010 3.81l-5.66 5.66a2.69 2.69 0 01-3.81 0l-.28-.28a2.69 2.69 0 010-3.81l5.66-5.66a2.69 2.69 0 013.81 0zM17.19 9.06l.28.28a2.69 2.69 0 010 3.81l-.28.28" />
                    </svg>
                    <p class="text-[14px] text-zinc-300 font-semibold">No call graph data</p>
                    <p class="text-[12px] text-zinc-500 mt-2 max-w-[280px] leading-relaxed">
                      Let the AI explore your codebase and map every function, type, interface, and module automatically.
                    </p>
                    <button
                      onClick={() => openBuildModal(false)}
                      class="mt-4 h-9 px-5 rounded-lg text-[13px] font-medium bg-[color:var(--accent)] text-[color:var(--on-primary)] hover:bg-[color:var(--accent-hover)] transition flex items-center gap-2"
                    >
                      <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M5 3l14 9-14 9V3z" />
                      </svg>
                      Build Call Graph
                    </button>
                  </>
                }>
                  <div class="w-10 h-10 border-2 border-[color:var(--accent)] border-t-transparent rounded-full animate-spin mb-4" />
                  <p class="text-[14px] text-zinc-300 font-semibold">Building call graph…</p>
                  <p class="text-[12px] text-zinc-500 mt-1.5 max-w-[260px]">
                    The AI is reading your source files and mapping all symbols and relationships.
                  </p>
                </Show>
              </div>
            </Show>

            <Show when={cg.loading()}>
              <div class="absolute inset-0 flex items-center justify-center bg-[color:var(--bg-base)]/80 z-10">
                <div class="flex flex-col items-center gap-3">
                  <div class="w-6 h-6 border-2 border-[color:var(--accent)] border-t-transparent rounded-full animate-spin" />
                  <p class="text-[12px] text-zinc-500">Loading call graph…</p>
                </div>
              </div>
            </Show>

            <Show when={cg.building() && filteredNodes().length > 0}>
              <div class="absolute top-3 left-1/2 -translate-x-1/2 z-20 flex items-center gap-2 bg-[color:var(--bg-surface)] border border-[color:var(--border-default)] rounded-full px-3 py-1.5 shadow-lg">
                <div class="w-3 h-3 border-2 border-[color:var(--accent)] border-t-transparent rounded-full animate-spin shrink-0" />
                <span class="text-[11px] text-zinc-400">Rebuilding call graph…</span>
              </div>
            </Show>

            <svg
              ref={svgRef}
              class="w-full h-full bg-grid"
              style={{ 'background-color': 'var(--bg-base)' }}
            />

            {/* Legend */}
            <Show when={filteredNodes().length > 0}>
              <div class="absolute bottom-3 left-3 flex items-center gap-3 bg-[color:var(--bg-surface)]/90 backdrop-blur border border-[color:var(--border-subtle)] rounded-lg px-3 py-2 text-[10px] text-zinc-500">
                <span class="flex items-center gap-1">
                  <svg class="w-3 h-3" viewBox="0 0 12 12">
                    <circle cx="6" cy="6" r="4" fill="#52525b" />
                  </svg>
                  direct
                </span>
                <span class="flex items-center gap-1">
                  <svg class="w-3 h-3" viewBox="0 0 12 12">
                    <circle cx="6" cy="6" r="4" fill="#f97316" />
                  </svg>
                  dynamic
                </span>
                <span class="flex items-center gap-1">
                  <svg class="w-3 h-3" viewBox="0 0 12 12">
                    <circle cx="6" cy="6" r="4" fill="#8b5cf6" />
                  </svg>
                  interface
                </span>
                <span class="flex items-center gap-1">
                  <svg class="w-3 h-3" viewBox="0 0 12 12">
                    <circle cx="6" cy="6" r="4" fill="#14b8a6" />
                  </svg>
                  callback
                </span>
              </div>

              {/* Zoom controls */}
              <div class="absolute bottom-3 right-3 flex flex-col gap-1">
                <button
                  onClick={() => {
                    if (svgRef && zoomBehavior) {
                      d3.select(svgRef).transition().duration(300).call(zoomBehavior.scaleBy, 1.3);
                    }
                  }}
                  class="w-7 h-7 rounded-md bg-[color:var(--bg-surface)] border border-[color:var(--border-subtle)] text-zinc-500 hover:text-zinc-200 hover:border-[color:var(--border-default)] flex items-center justify-center transition"
                  title="Zoom in"
                >
                  <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
                  </svg>
                </button>
                <button
                  onClick={() => {
                    if (svgRef && zoomBehavior) {
                      d3.select(svgRef).transition().duration(300).call(zoomBehavior.scaleBy, 0.7);
                    }
                  }}
                  class="w-7 h-7 rounded-md bg-[color:var(--bg-surface)] border border-[color:var(--border-subtle)] text-zinc-500 hover:text-zinc-200 hover:border-[color:var(--border-default)] flex items-center justify-center transition"
                  title="Zoom out"
                >
                  <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M20 12H4" />
                  </svg>
                </button>
                <button
                  onClick={() => {
                    if (svgRef && zoomBehavior) {
                      d3.select(svgRef).transition().duration(500).call(zoomBehavior.transform, d3.zoomIdentity);
                    }
                  }}
                  class="w-7 h-7 rounded-md bg-[color:var(--bg-surface)] border border-[color:var(--border-subtle)] text-zinc-500 hover:text-zinc-200 hover:border-[color:var(--border-default)] flex items-center justify-center transition"
                  title="Reset view"
                >
                  <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                  </svg>
                </button>
              </div>
            </Show>
          </div>

          {/* Detail panel */}
          <div class="w-[320px] shrink-0 border-l border-[color:var(--border-subtle)] bg-[color:var(--bg-surface)] flex flex-col overflow-hidden">
            {/* Panel header */}
            <div class="shrink-0 px-4 py-2.5 border-b border-[color:var(--border-subtle)] flex items-center justify-between">
              <span class="text-[12px] font-semibold text-zinc-300">Node Details</span>
              <Show when={selectedDetail()}>
                <button
                  onClick={handleResetView}
                  class="w-5 h-5 flex items-center justify-center text-zinc-500 hover:text-zinc-200 transition"
                  title="Close detail"
                >
                  <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </Show>
            </div>

            <Show when={detailLoading()}>
              <div class="flex-1 flex items-center justify-center">
                <div class="w-5 h-5 border-2 border-[color:var(--accent)] border-t-transparent rounded-full animate-spin" />
              </div>
            </Show>

            <Show when={!detailLoading()}>
              <NodeDetailPanel
                detail={selectedDetail()}
                loading={detailLoading()}
                onSelectNode={handleSelectNode}
              />
            </Show>
          </div>
        </div>
      </div>

      {/* Build Call Graph Modal */}
      <Show when={showBuildModal()}>
        {/* Backdrop */}
        <div
          class="fixed inset-0 z-50 bg-black/60 backdrop-blur-[2px] flex items-center justify-center p-4"
          onClick={(e) => { if (e.target === e.currentTarget) setShowBuildModal(false); }}
        >
          {/* Dialog */}
          <div class="w-full max-w-[440px] bg-[color:var(--bg-surface)] border border-[color:var(--border-default)] rounded-2xl shadow-[0_24px_64px_rgba(0,0,0,0.6)] flex flex-col overflow-hidden">

            {/* Header */}
            <div class="px-6 pt-6 pb-4 border-b border-[color:var(--border-subtle)]">
              <div class="flex items-start justify-between gap-3">
                <div>
                  <h2 class="text-[16px] font-semibold text-zinc-100">
                    {modalIsRebuild() ? 'Rebuild Call Graph' : 'Build Call Graph'}
                  </h2>
                  <p class="text-[12px] text-zinc-500 mt-1 leading-relaxed">
                    {modalIsRebuild()
                      ? 'Choose a model to re-analyze your codebase from scratch.'
                      : 'Choose a model to explore your codebase and map every symbol and relationship.'}
                  </p>
                </div>
                <button
                  onClick={() => setShowBuildModal(false)}
                  class="w-7 h-7 rounded-lg flex items-center justify-center text-zinc-500 hover:text-zinc-200 hover:bg-[color:var(--bg-elevated)] transition shrink-0 mt-0.5"
                >
                  <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>

            {/* Model list */}
            <div class="flex-1 overflow-y-auto px-3 py-3 max-h-[360px]">
              <Show when={cg.models().filter(m => m.enabled).length === 0}>
                <div class="px-3 py-8 text-center text-[12px] text-zinc-500">
                  No models available. Configure a provider in Settings.
                </div>
              </Show>
              <For each={
                [...new Set(cg.models().filter(m => m.enabled).map(m => m.providerId))]
              }>
                {(providerId) => {
                  const providerModels = () => cg.models().filter(m => m.enabled && m.providerId === providerId);
                  const providerLabel: Record<string, string> = {
                    anthropic: 'Anthropic', openai: 'OpenAI', openrouter: 'OpenRouter',
                    google: 'Google', mistral: 'Mistral',
                  };
                  const providerColor: Record<string, string> = {
                    anthropic: '#fb923c', openai: '#34d399', openrouter: '#a78bfa',
                    google: '#60a5fa', mistral: '#f43f5e',
                  };
                  return (
                    <div class="mb-1">
                      <div class="flex items-center gap-2 px-2 py-1.5 mb-0.5">
                        <span class="w-1.5 h-1.5 rounded-full shrink-0" style={{ background: providerColor[providerId] || '#71717a' }} />
                        <span class="text-[10px] font-semibold uppercase tracking-wider" style={{ color: providerColor[providerId] || '#71717a' }}>
                          {providerLabel[providerId] || providerId}
                        </span>
                      </div>
                      <For each={providerModels()}>
                        {(model) => {
                          const isSelected = () => cg.selectedModel() === model.id;
                          return (
                            <button
                              onClick={() => cg.selectModel(model.id)}
                              class={`w-full text-left px-3 py-2.5 rounded-lg mb-0.5 flex items-center justify-between gap-3 transition-colors
                                ${isSelected()
                                  ? 'bg-[color:var(--accent-soft)] text-[color:var(--accent)]'
                                  : 'text-zinc-200 hover:bg-[color:var(--bg-elevated)]'
                                }`}
                            >
                              <div class="flex items-center gap-2.5 min-w-0">
                                <div class={`w-4 h-4 rounded-full border-2 flex items-center justify-center shrink-0 transition-colors
                                  ${isSelected() ? 'border-[color:var(--accent)] bg-[color:var(--accent)]' : 'border-zinc-600'}`}
                                >
                                  <Show when={isSelected()}>
                                    <svg class="w-2.5 h-2.5 text-white" fill="currentColor" viewBox="0 0 24 24">
                                      <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41L9 16.17z" />
                                    </svg>
                                  </Show>
                                </div>
                                <span class="text-[13px] font-medium truncate">{model.name}</span>
                                <Show when={model.default}>
                                  <span class="text-[9px] text-zinc-500 uppercase tracking-wider shrink-0">default</span>
                                </Show>
                              </div>
                              <Show when={model.inputPricePerM > 0}>
                                <span class="text-[10px] text-zinc-500 font-mono shrink-0">
                                  ${model.inputPricePerM % 1 === 0 ? model.inputPricePerM : model.inputPricePerM.toFixed(2)}
                                  <span class="text-zinc-700">/</span>
                                  ${model.outputPricePerM % 1 === 0 ? model.outputPricePerM : model.outputPricePerM.toFixed(2)}
                                </span>
                              </Show>
                            </button>
                          );
                        }}
                      </For>
                    </div>
                  );
                }}
              </For>
            </div>

            {/* Footer */}
            <div class="px-6 py-4 border-t border-[color:var(--border-subtle)] flex items-center justify-end gap-3">
              <button
                onClick={() => setShowBuildModal(false)}
                class="h-9 px-4 rounded-lg text-[13px] text-zinc-400 hover:text-zinc-200 hover:bg-[color:var(--bg-elevated)] transition"
              >
                Cancel
              </button>
              <button
                onClick={handleConfirmBuild}
                disabled={!cg.selectedModel()}
                class="h-9 px-5 rounded-lg text-[13px] font-medium bg-[color:var(--accent)] text-[color:var(--on-primary)] hover:bg-[color:var(--accent-hover)] disabled:opacity-50 disabled:cursor-not-allowed transition flex items-center gap-2"
              >
                <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M5 3l14 9-14 9V3z" />
                </svg>
                {modalIsRebuild() ? 'Rebuild' : 'Build Call Graph'}
              </button>
            </div>
          </div>
        </div>
      </Show>
    </div>
  );
}