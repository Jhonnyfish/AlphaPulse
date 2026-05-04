import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'

export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
    ],
    languageOptions: {
      globals: globals.browser,
    },
    rules: {
      // React 19 strict rules — downgrade to warn (common patterns that work fine)
      'react-hooks/set-state-in-effect': 'warn',       // setState after fetch in useEffect
      'react-hooks/static-components': 'warn',          // helper components defined in render
      'react-hooks/refs': 'warn',                       // ref access for ECharts/imperative APIs
      'react-hooks/purity': 'warn',                     // deterministic mock data in useMemo
      'react-refresh/only-export-components': 'warn',   // shared context files (auth, theme, toast)
    },
  },
])
