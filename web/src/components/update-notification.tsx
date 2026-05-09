import { createSignal, createEffect, Show, onMount } from 'solid-js';
import { getVersion, checkForUpdate, type VersionResponse } from '../api/client';

export default function UpdateNotification() {
  const [versionInfo, setVersionInfo] = createSignal<VersionResponse | null>(null);
  const [isChecking, setIsChecking] = createSignal(false);
  const [isVisible, setIsVisible] = createSignal(true);
  const [dismissedUntil, setDismissedUntil] = createSignal<number>(0);
  const [isHovered, setIsHovered] = createSignal(false);

  // Load dismissed state from localStorage
  onMount(() => {
    const stored = localStorage.getItem('ogcode-update-dismissed-until');
    if (stored) {
      setDismissedUntil(parseInt(stored, 10));
    }
  });

  // Check for updates on mount
  onMount(() => {
    checkForUpdateCheck();
  });

  async function checkForUpdateCheck() {
    try {
      const info = await getVersion();
      setVersionInfo(info);

      // Auto-hide if no update available or already dismissed
      const now = Date.now();
      if (!info.updateAvailable || now < dismissedUntil()) {
        setIsVisible(false);
      }
    } catch (err) {
      console.error('Failed to check version:', err);
    }
  }

  async function handleCheckUpdate() {
    setIsChecking(true);
    try {
      const info = await checkForUpdate();
      if (!info.updateAvailable) {
        setIsVisible(false);
      } else {
        // Refresh version info
        const fullInfo = await getVersion();
        setVersionInfo(fullInfo);
      }
    } catch (err) {
      console.error('Failed to check update:', err);
    } finally {
      setIsChecking(false);
    }
  }

  function handleDismiss() {
    setIsVisible(false);
    // Dismiss for 24 hours
    const dismissUntil = Date.now() + 24 * 60 * 60 * 1000;
    setDismissedUntil(dismissUntil);
    localStorage.setItem('ogcode-update-dismissed-until', dismissUntil.toString());
  }

  function handleDismissPermanent() {
    setIsVisible(false);
    // Dismiss for 7 days
    const dismissUntil = Date.now() + 7 * 24 * 60 * 60 * 1000;
    setDismissedUntil(dismissUntil);
    localStorage.setItem('ogcode-update-dismissed-until', dismissUntil.toString());
  }

  function formatDate(dateStr: string): string {
    if (!dateStr) return '';
    const date = new Date(dateStr);
    return date.toLocaleDateString();
  }

  function getPlatformIcon(command: string): string {
    if (command.includes('winget')) return '⊞';
    if (command.includes('brew')) return '🍺';
    if (command.includes('scoop')) return '🍨';
    if (command.includes('cargo')) return '🦀';
    return '⬆️';
  }

  // Don't show if shouldHide
  const shouldHide = () => !isVisible() || !versionInfo()?.updateAvailable;

  return (
    <Show when={!shouldHide()}>
      <div
        class="fixed bottom-4 right-4 z-50 max-w-lg transition-all duration-300"
        classList={{
          'translate-y-0 opacity-100': !shouldHide(),
          'translate-y-4 opacity-0': shouldHide(),
        }}
        onMouseEnter={() => setIsHovered(true)}
        onMouseLeave={() => setIsHovered(false)}
      >
        <div class="bg-zinc-800/95 backdrop-blur-sm border border-zinc-700 rounded-lg shadow-2xl overflow-hidden">
          {/* Header */}
          <div class="flex items-center justify-between px-4 py-3 bg-gradient-to-r from-blue-600/20 to-purple-600/20 border-b border-zinc-700/50">
            <div class="flex items-center gap-2">
              <span class="text-lg">🎉</span>
              <span class="font-semibold text-white">Update Available</span>
            </div>
            <div class="flex items-center gap-1">
              <span class="text-xs text-zinc-400">v{versionInfo()?.version} → v{versionInfo()?.latestVersion?.replace(/^v/, '')}</span>
              <button
                onClick={handleDismiss}
                class="p-1 hover:bg-zinc-700 rounded transition-colors ml-2"
                title="Dismiss for 24 hours"
              >
                <svg class="w-4 h-4 text-zinc-400 hover:text-zinc-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                </svg>
              </button>
            </div>
          </div>

          {/* Content */}
          <div class="p-4 space-y-3">
            {/* Release notes (collapsed by default) */}
            <Show when={versionInfo()?.releaseNotes}>
              <div class="text-sm text-zinc-300 bg-zinc-900/50 rounded p-3">
                <div class="font-medium text-zinc-400 text-xs uppercase tracking-wider mb-2">What's New</div>
                <div class="prose prose-invert prose-sm max-w-none">
                  {versionInfo()?.releaseNotes?.substring(0, 200)}
                  {(versionInfo()?.releaseNotes?.length || 0) > 200 ? '...' : ''}
                </div>
                <a
                  href={versionInfo()?.releaseUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  class="inline-flex items-center gap-1 text-blue-400 hover:text-blue-300 text-xs mt-2"
                >
                  View full release notes
                  <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/>
                  </svg>
                </a>
              </div>
            </Show>

            {/* Install command */}
            <Show when={versionInfo()?.installCommand}>
              <div class="bg-zinc-900/80 rounded p-3 border border-zinc-700/50">
                <div class="flex items-center justify-between mb-2">
                  <span class="text-xs text-zinc-500 uppercase tracking-wider flex items-center gap-1">
                    <span>{getPlatformIcon(versionInfo()?.installCommand || '')}</span>
                    Install Command
                  </span>
                  <button
                    onClick={() => {
                      navigator.clipboard.writeText(versionInfo()?.installCommand || '');
                    }}
                    class="text-xs text-zinc-400 hover:text-white flex items-center gap-1 transition-colors"
                    title="Copy to clipboard"
                  >
                    <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                    </svg>
                    Copy
                  </button>
                </div>
                <code class="text-sm text-green-400 font-mono block">
                  {versionInfo()?.installCommand}
                </code>
              </div>
            </Show>
          </div>

          {/* Footer */}
          <div class="px-4 py-3 bg-zinc-900/50 border-t border-zinc-700/50 flex items-center justify-between">
            <button
              onClick={handleDismissPermanent}
              class="text-xs text-zinc-500 hover:text-zinc-400 transition-colors"
            >
              Don't show again (7 days)
            </button>
            <button
              onClick={handleCheckUpdate}
              disabled={isChecking()}
              class="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-500 disabled:bg-zinc-700 disabled:cursor-not-allowed text-white text-sm rounded transition-all"
            >
              <Show when={isChecking()}>
                <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"/>
                </svg>
              </Show>
              <Show when={!isChecking()}>
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                </svg>
              </Show>
              {isChecking() ? 'Checking...' : 'Check Again'}
            </button>
          </div>
        </div>
      </div>
    </Show>
  );
}
