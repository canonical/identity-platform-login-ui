import { Graph } from '../src/core/graph';
import { loginRules } from '../src/heuristics/rules/login';
import { recoveryRules } from '../src/heuristics/rules/recovery';
import { webauthnRules } from '../src/heuristics/rules/webauthn';
import { settingsRules } from '../src/heuristics/rules/settings';
import { negativeRules } from '../src/heuristics/rules/negative';
import chalk from 'chalk';

// Mock services since we just want the graph structure
const mockWebAuthn: any = {};

async function main() {
    const graph = new Graph();
    
    // Load all rules
    loginRules(graph);
    recoveryRules(graph);
    webauthnRules(graph, mockWebAuthn);
    settingsRules(graph);
    negativeRules(graph);

    // Extract Graph Data
    const states = Array.from((graph as any).states.values()).map((s: any) => s.name);
    const transitions = (graph as any).transitions.map((t: any) => ({
        from: t.from,
        to: t.to,
        trigger: t.trigger.description
    }));

    console.log(chalk.bold("🤖 QA Persona: Coverage Analyst"));
    console.log("----------------------------------------------------------------");
    console.log("I have dumped the current Heuristic Graph. Ask Copilot what's missing.");
    console.log("Copy the block below:");
    console.log("----------------------------------------------------------------");
    console.log("");
    console.log("I am building a Monkey Testing Agent for my Login UI.");
    console.log("Here is the current State Machine graph it supports:");
    console.log("");
    console.log("States:", JSON.stringify(states));
    console.log("Transitions:", JSON.stringify(transitions));
    console.log("");
    console.log("Considering this is an OIDC Provider (Login, MFA, Recovery), what critical user flows are missing?");
    console.log("Suggest 3 new heuristic rules I should add.");
    console.log("----------------------------------------------------------------");
}

main();
