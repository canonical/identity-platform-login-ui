import { AgentState, Transition } from './types';

export class Graph {
  private states: Map<string, AgentState> = new Map();
  private transitions: Transition[] = [];
  private currentStateId: string | null = null;

  constructor(initialStates: AgentState[] = []) {
    initialStates.forEach(s => this.states.set(s.id, s));
  }

  addState(state: AgentState) {
    this.states.set(state.id, state);
  }

  addTransition(transition: Transition) {
    this.transitions.push(transition);
  }

  getPossibleTransitions(stateId: string): Transition[] {
    return this.transitions.filter(t => t.from === stateId);
  }

  setState(stateId: string) {
    if (!this.states.has(stateId)) {
      throw new Error(`State ${stateId} not found in graph`);
    }
    this.currentStateId = stateId;
  }

  getCurrentState(): AgentState | undefined {
    return this.currentStateId ? this.states.get(this.currentStateId) : undefined;
  }

  // Find state matching current URL - check more specific patterns first
  async matchState(url: string, page?: any): Promise<AgentState | undefined> {
    // Sort states by pattern specificity (longer patterns = more specific)
    const sortedStates = Array.from(this.states.values()).sort((a, b) => {
      const aLen = a.urlPattern.source.length;
      const bLen = b.urlPattern.source.length;
      return bLen - aLen; // Descending order - check longest patterns first
    });
    
    for (const state of sortedStates) {
      if (state.urlPattern.test(url)) {
        // If state has a contentCheck, validate it
        if (state.contentCheck && page) {
          try {
            const matches = await state.contentCheck(page);
            if (matches) {
              console.log(`[Graph] Matched state "${state.name}" for URL: ${url} (via content check)`);
              return state;
            }
            // Content check failed, try next state
            continue;
          } catch (error) {
            console.log(`[Graph] Content check failed for "${state.name}": ${error}`);
            continue;
          }
        }
        
        // No content check or no page provided, use URL pattern only
        console.log(`[Graph] Matched state "${state.name}" for URL: ${url} (via URL pattern)`);
        return state;
      }
    }
    console.log(`[Graph] No state matched for URL: ${url}`);
    return undefined;
  }
}
