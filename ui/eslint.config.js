const {
    defineConfig,
} = require("eslint/config");

const tsParser = require("@typescript-eslint/parser");
const typescriptEslint = require("@typescript-eslint/eslint-plugin");
const react = require("eslint-plugin-react");
const globals = require("globals");
const js = require("@eslint/js");

const {
    FlatCompat,
} = require("@eslint/eslintrc");

const compat = new FlatCompat({
    baseDirectory: __dirname,
    recommendedConfig: js.configs.recommended,
    allConfig: js.configs.all
});

module.exports = defineConfig([
    {
        ignores: [
            "**/node_modules/**",
            "dist/**",
            ".next/**",
            "playwright-report/**",
            "test-results/**",
            "next.config.js",
            "eslint.config.js",
            ".eslintrc.js",
            "next-env.d.ts",
        ],
    },
    {
        languageOptions: {
            parser: tsParser,

        globals: {
            ...globals.node,
            ...globals.browser,
            ...globals.jest,
        },

        sourceType: "module",

        parserOptions: {
            project: "./tsconfig.json",

            ecmaFeatures: {
                jsx: true,
            },
        },
    },

    plugins: {
        "@typescript-eslint": typescriptEslint,
        react,
    },

    extends: compat.extends(
        "plugin:@next/next/recommended",
        "eslint:recommended",
        "plugin:react/recommended",
        "plugin:prettier/recommended",
        "plugin:@typescript-eslint/recommended",
        "plugin:@typescript-eslint/recommended-requiring-type-checking",
        "plugin:@typescript-eslint/strict",
    ),

    settings: {
        react: {
            version: "detect",
        },
    },

    rules: {
        "linebreak-style": ["error", "unix"],
        semi: ["error", "always"],
        "object-curly-spacing": ["error", "always"],

        "@typescript-eslint/no-unused-vars": ["error", {
            argsIgnorePattern: "^_",
        }],

        "@typescript-eslint/no-explicit-any": "off",
        "@typescript-eslint/no-unsafe-assignment": "off",
        "@typescript-eslint/no-unsafe-member-access": "off",
        "@typescript-eslint/no-redundant-type-constituents": "off",
    },
}]);
