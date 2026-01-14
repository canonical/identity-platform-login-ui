import { Page, ElementHandle } from 'playwright';

export interface AgentState {
  id: string;
  urlPattern: RegExp;
  name: string;
  contentCheck?: (page: Page) => Promise<boolean>;  // Optional content-based validation
}

export interface Transition {
  from: string;
  to: string;
  trigger: HeuristicRule;
}

export interface HeuristicRule {
  id: string;
  description: string;
  condition: (page: Page) => Promise<boolean>;
  action: (page: Page) => Promise<void>;
  weight?: number; // For exploration mode
}

export interface ChaosConfig {
  mode: 'validation' | 'exploration';
  baseUrl: string;
  duration?: number;
  matrix?: {
    identifierFirst: boolean;
    mfaEnabled: boolean;
  };
}

export interface ScanResult {
    interactives: {
        role: string;
        selector: string;
        text?: string;
    }[];
    forms: {
        selector: string;
        inputs: string[];
    }[];
}
