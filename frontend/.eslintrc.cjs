module.exports = {
  root: true,
  env: { browser: true, es2020: true },
  extends: [
    'eslint:recommended',
    'plugin:react/recommended',
    'plugin:react/jsx-runtime',
    'plugin:react-hooks/recommended',
  ],
  ignorePatterns: ['dist', '.eslintrc.cjs'],
  parserOptions: { ecmaVersion: 'latest', sourceType: 'module' },
  settings: { react: { version: '18.2' } },
  plugins: ['react-refresh'],
  rules: {
    'react-refresh/only-export-components': [
      'warn',
      { allowConstantExport: true },
    ],
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
          "Literal[value=/(?:^|[\\s\"'`])(?:hover:|focus:|focus-visible:|group-hover:|active:|disabled:)?(?:text|bg|border|ring|divide|fill|stroke|from|to|via)-(?:red|emerald|amber|stone|sky|blue|rose|teal|cyan|indigo|orange|green|gray|zinc|slate|neutral|lime|yellow|violet|purple|fuchsia|pink)-[0-9]/]",
        message:
          'Raw Tailwind palette class — use the semantic design tokens instead (see DESIGN.md). Chart/severity ramps may disable this rule per-line with a justification.',
      },
    ],
  },
  overrides: [
    {
      // Build/config files run in Node, not the browser.
      files: ['*.config.js', 'vite.config.js', 'tailwind.config.js'],
      env: { node: true, browser: false },
    },
    {
      // Test files run under vitest/jsdom with Node globals available.
      files: ['**/*.test.{js,jsx}', '**/__tests__/**'],
      env: { node: true },
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
  ],
}
