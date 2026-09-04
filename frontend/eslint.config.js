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
      // This app does not run the compiler. Two of those rules flag
      // deliberate, documented idioms rather than bugs, and are relaxed
      // NARROWLY rather than globally, so a new file can't adopt them unseen:
      //  - set-state-in-effect: off only for the files listed in the override
      //    below (the "copy loaded server data into local form state" pattern).
      //  - refs: the "latest ref" idiom (`ref.current = value` during render)
      //    stays ON; each site carries a `// eslint-disable-next-line
      //    react-hooks/refs -- reason` (AuthProvider, useUrlState, VerifyEmail,
      //    DataTable's new-row reveal).
      // Revisit both if the React Compiler is ever adopted.
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
    // The "copy loaded server data into local state" idiom
    // (`useEffect(() => { if (data) setForm(data) }, [data])`) on settings
    // pages, slide-overs and a few list pages. Listed per file: a file not on
    // this list gets the rule at full strength, so add to the list (with a
    // reason in the PR) rather than re-widening the rule.
    files: [
      'src/components/IntegrationConnections.jsx',
      'src/components/PaymentGateways.jsx',
      'src/components/patterns/MotionState.jsx',
      'src/components/slide-overs/CancelFlowDetail.jsx',
      'src/components/slide-overs/CustomerDetail.jsx',
      'src/components/slide-overs/DunningCampaignDetail.jsx',
      'src/components/slide-overs/PlanCharges.jsx',
      'src/components/slide-overs/PlanDetail.jsx',
      'src/components/ui/command-palette.jsx',
      'src/lib/useReducedMotion.js',
      'src/pages/AskAnalytics.jsx',
      'src/pages/CreateQuote.jsx',
      'src/pages/Dashboard.jsx',
      'src/pages/FinanceReconciliation.jsx',
      'src/pages/Ledger.jsx',
      'src/pages/Login.jsx',
      'src/pages/Security.jsx',
      'src/pages/Settings.jsx',
      'src/pages/SubscriptionPage.jsx',
      'src/pages/portal/PortalDashboard.jsx',
      'src/pages/portal/PortalPaymentMethod.jsx',
      'src/pages/settings/EUEInvoiceSettings.jsx',
      'src/pages/settings/GSTSettings.jsx',
      'src/pages/settings/IRPSettings.jsx',
      'src/pages/settings/InvoiceBranding.jsx',
      'src/pages/settings/MCPSettings.jsx',
      'src/pages/settings/TaxNexusSettings.jsx',
      'src/pages/settings/USTaxSettings.jsx',
    ],
    rules: { 'react-hooks/set-state-in-effect': 'off' },
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
