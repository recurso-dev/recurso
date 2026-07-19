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
    'react/no-unescaped-entities': 'off'
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
