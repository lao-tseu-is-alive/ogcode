import { createContext, useContext, createSignal, type ParentComponent } from 'solid-js';
import { getProviderConfigs, type ProviderConfig } from '../api/client';

interface OnboardingContextValue {
  // True once the initial provider-config check has completed.
  loaded: () => boolean;
  // True when no LLM provider is configured (neither via env var nor saved to
  // the DB) — i.e. the user should be sent through the onboarding wizard.
  needsOnboarding: () => boolean;
  // True when the user chose to skip onboarding for this session. The redirect
  // gate honours this so "Skip for now" doesn't bounce straight back.
  dismissed: () => boolean;
  // Mark onboarding as skipped for this session.
  dismiss: () => void;
  // Re-run the check (e.g. after the wizard saves credentials).
  refresh: () => Promise<void>;
}

const OnboardingContext = createContext<OnboardingContextValue>();

// A provider counts as "configured" when its key is stored in the DB (apiKey
// sentinel "__SET__") or supplied via environment variable. Keyless local
// providers (Ollama) instead count as configured once a base URL is set —
// otherwise completing the wizard with Ollama would leave the app looking
// unconfigured and bounce the user back into onboarding.
function isConfigured(configs: ProviderConfig[]): boolean {
  return configs.some(
    (c) =>
      c.apiKey === '__SET__' ||
      c.envKeySet ||
      (c.providerId === 'ollama' && (c.baseUrl !== '' || c.envBaseURLSet)),
  );
}

export const OnboardingProvider: ParentComponent = (props) => {
  const [loaded, setLoaded] = createSignal(false);
  const [needsOnboarding, setNeedsOnboarding] = createSignal(false);
  const [dismissed, setDismissed] = createSignal(false);

  const refresh = async () => {
    try {
      const configs = await getProviderConfigs();
      setNeedsOnboarding(!isConfigured(configs));
    } catch {
      // Fail open: if the check errors, never trap the user in onboarding.
      setNeedsOnboarding(false);
    } finally {
      setLoaded(true);
    }
  };

  const dismiss = () => setDismissed(true);

  void refresh();

  const value: OnboardingContextValue = { loaded, needsOnboarding, dismissed, dismiss, refresh };
  return (
    <OnboardingContext.Provider value={value}>
      {props.children}
    </OnboardingContext.Provider>
  );
};

export function useOnboarding() {
  const ctx = useContext(OnboardingContext);
  if (!ctx) throw new Error('useOnboarding must be used within OnboardingProvider');
  return ctx;
}
