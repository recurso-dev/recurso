// ESLint 9 flat config — the one-to-one port of the old .eslintrc.cjs.
// Run: npm run lint (eslint . --report-unused-disable-directives --max-warnings 0)
import js from '@eslint/js'
import globals from 'globals'
import react from 'eslint-plugin-react'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'

export default [
  { ignores: ['dist', 'node_modules'] },

  js.configs.recommended,
  react.configs.flat.recommended,
  react.configs.flat['jsx-runtime'],
  reactHooks.configs.flat.recommended,

  {
    files: ['**/*.{js,jsx}'],
    languageOptions: {
      ecmaVersion: 'latest',
      sourceType: 'module',
      parserOptions: { ecmaFeatures: { jsx: true } },
      globals: { ...globals.browser, ...globals.es2020 },
    },
    settings: { react: { version: '19' } },
    plugins: { 'react-refresh': reactRefresh },
    rules: {
      'react-refresh/only-export-components': [
        'warn',
        { allowConstantExport: true },
      ],
      // eslint-plugin-react-hooks 6+/7 ships the React Compiler rule set.
      // This app does not run the compiler, and two of those rules flag
      // deliberate, documented idioms used throughout the codebase rather
      // than bugs — so they are switched off here instead of mass-editing:
      //  - set-state-in-effect: the "copy loaded server data into local form
      //    state" pattern (`useEffect(() => { if (data) setForm(data) }, [data])`)
      //    on every settings page and slide-over (30 sites).
      //  - refs: the "latest ref" idiom (`ref.current = value` during render)
      //    in AuthProvider, useUrlState, VerifyEmail and DataTable's
      //    new-row reveal — each with an inline comment on why.
      // Revisit both if the React Compiler is ever adopted.
      'react-hooks/set-state-in-effect': 'off',
      'react-hooks/refs': 'off',
      'react/prop-types': 'off', // specific to this project preferences
      'react/no-unescaped-entities': 'off',
      // Design-system guard (DASHBOARD_REDESIGN.md Stage 1): raw Tailwind
      // palette classes are banned — use the semantic tokens (primary, success,
      // warning, info, destructive, muted, subtle, foreground, border, canvas).
      // Data-visualization ramps (chart series palettes, severity scales) are
      // the one sanctioned exception — mark them with an eslint-disable comment
      // that says WHY.
      'no-restricted-syntax': [
        'error',
        {
          selector:
            "Literal[value=/(?:^|[\\s\"'`])(?:hover:|focus:|focus-visible:|group-hover:|active:|disabled:)?(?:text|bg|border|ring|divide|fill|stroke|from|to|via|accent|caret|outline|decoration|shadow)-(?:red|emerald|amber|stone|sky|blue|rose|teal|cyan|indigo|orange|green|gray|zinc|slate|neutral|lime|yellow|violet|purple|fuchsia|pink)-[0-9]/]",
          message:
            'Raw Tailwind palette class — use the semantic design tokens instead (see DESIGN.md). Chart/severity ramps may disable this rule per-line with a justification.',
        },
      ],
    },
  },

  {
    // Build/config files run in Node, not the browser.
    files: ['*.config.js', 'vite.config.js', 'tailwind.config.js'],
    languageOptions: { globals: { ...globals.node } },
  },

  {
    // Test files run under vitest/jsdom with Node globals available.
    files: ['**/*.test.{js,jsx}', '**/__tests__/**'],
    languageOptions: { globals: { ...globals.browser, ...globals.node } },
  },

  {
    // shadcn UI primitives and context providers intentionally co-locate a
    // component with a variants constant / hook; fast-refresh warning is moot.
    files: [
      'src/components/ui/**',
      'src/auth/AuthProvider.jsx',
      'src/components/Toast.jsx',
    ],
    rules: { 'react-refresh/only-export-components': 'off' },
  },
]
