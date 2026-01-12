# Copilot Instructions - Documentation Structure

This repository uses a **modular approach** to Copilot instructions to avoid token limit issues and improve maintainability.

## Structure

### Main Instructions File
**`.github/copilot-instructions.md`**
- Core project overview and quick reference
- Critical coding rules (non-negotiable)
- Essential commands and architecture patterns
- References to detailed guides in `.github/instructions/`

### Supporting Documentation
**`.github/instructions/`** (Automatically loaded by Copilot CLI)
- **`go.instructions.md`** - Complete Go standards (error handling, naming, testing, etc.)
- **`react.instructions.md`** - TypeScript/React standards (components, hooks, testing)
- **`workflows.instructions.md`** - Step-by-step guides for common tasks

## Benefits of This Approach

1. **Automatic Context Loading**: The `.instructions.md` suffix ensures GitHub Copilot CLI automatically indexes these files.
2. **Token Efficiency**: Main file stays concise, avoiding context window issues.
3. **Maintainability**: Detailed examples separated from quick reference.
4. **Focus**: AI gets critical rules first, can reference details when needed.

## Usage

When using GitHub Copilot or other AI assistants:
- The main `.github/copilot-instructions.md` is automatically loaded.
- The specific files in `.github/instructions/` are also automatically loaded into the context when relevant (or globally, depending on CLI version).
- All files follow the same rigor and standards as `canonical/hook-service`.

## Updating Instructions

When adding new patterns or conventions:
1. Add concise reference to main `copilot-instructions.md`.
2. Add detailed examples/explanations to appropriate file in `.github/instructions/`.
3. Update this summary if adding new supporting docs.
