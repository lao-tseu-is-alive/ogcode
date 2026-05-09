export interface ProviderDef {
  id: string;
  label: string;
  dot: string;
  bg: string;
  ring: string;
  hasBaseURL: boolean;
  supportsEmbed: boolean;
}

export const PROVIDER_DEFS: ProviderDef[] = [
  { id: 'anthropic',  label: 'Anthropic',  dot: 'bg-orange-400',  bg: 'bg-orange-500/10',  ring: 'ring-orange-400/20', hasBaseURL: false, supportsEmbed: false },
  { id: 'openai',     label: 'OpenAI',     dot: 'bg-emerald-400', bg: 'bg-emerald-500/10', ring: 'ring-emerald-400/20', hasBaseURL: true,  supportsEmbed: true  },
  { id: 'openrouter', label: 'OpenRouter', dot: 'bg-violet-400',  bg: 'bg-violet-500/10',  ring: 'ring-violet-400/20', hasBaseURL: false, supportsEmbed: true  },
  { id: 'ollama',     label: 'Ollama',     dot: 'bg-sky-400',     bg: 'bg-sky-500/10',     ring: 'ring-sky-400/20',    hasBaseURL: true,  supportsEmbed: true  },
];

export const EMBED_PROVIDERS = PROVIDER_DEFS.filter((p) => p.supportsEmbed);

export const CHAT_PROVIDERS = [
  { id: '', label: 'Use default (your main LLM)' },
  ...PROVIDER_DEFS.map((p) => ({ id: p.id, label: p.label })),
];
