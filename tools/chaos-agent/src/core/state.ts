import * as fs from 'fs';
import * as path from 'path';

/**
 * Global state manager for chaos agent
 * Stores runtime state like TOTP secrets across different phases
 * Persists critical data to disk for use across test runs
 */
class AgentState {
    private state: Map<string, any>;
    private persistPath: string;

    constructor() {
        this.state = new Map();
        this.persistPath = path.join(process.cwd(), '.chaos-agent-state.json');
        this.load();
    }

    set(key: string, value: any): void {
        this.state.set(key, value);

        // Auto-persist certain keys
        if (key === 'totpSecret' || key === 'secrets' || key === 'users') {
            this.persist();
        }
    }

    get<T>(key: string): T | undefined {
        return this.state.get(key);
    }

    has(key: string): boolean {
        return this.state.has(key);
    }

    delete(key: string): void {
        this.state.delete(key);
        this.persist();
    }

    clear(): void {
        this.state.clear();
        this.persist();
    }

    /**
     * Load persisted state from disk
     */
    private load(): void {
        try {
            if (fs.existsSync(this.persistPath)) {
                const data = fs.readFileSync(this.persistPath, 'utf-8');
                const parsed = JSON.parse(data);
                Object.entries(parsed).forEach(([key, value]) => {
                    this.state.set(key, value);
                });
                console.log('[AgentState] Loaded persisted state from disk');
            }
        } catch (error) {
            console.log('[AgentState] Could not load persisted state:', error);
        }
    }

    /**
     * Persist state to disk
     */
    private persist(): void {
        try {
            const data: Record<string, any> = {};
            this.state.forEach((value, key) => {
                data[key] = value;
            });
            fs.writeFileSync(this.persistPath, JSON.stringify(data, null, 2), 'utf-8');
            console.log('[AgentState] Persisted state to disk');
        } catch (error) {
            console.log('[AgentState] Could not persist state:', error);
        }
    }
}

// Singleton instance
export const agentState = new AgentState();
