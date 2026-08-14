/** @type {import('tailwindcss').Config} */
import colors from "tailwindcss/colors";

export default {
  // Light-only enterprise theme. Dark mode is intentionally not configured —
  // `dark:` variants must never be written (DESIGN.md §3) and no longer compile.
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
    // Tremor needs its own source scanned so its utility classes are generated.
    "./node_modules/@tremor/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    container: {
      center: true,
      padding: "2rem",
      screens: { "2xl": "1400px" },
    },
    extend: {
      colors: {
        // ---- shadcn/ui semantic tokens (driven by CSS vars in index.css) ----
        border: "hsl(var(--border))",
        input: "hsl(var(--input))",
        ring: "hsl(var(--ring))",
        background: "hsl(var(--background))",
        canvas: "hsl(var(--canvas))",
        foreground: "hsl(var(--foreground))",
        primary: {
          DEFAULT: "hsl(var(--primary))",
          foreground: "hsl(var(--primary-foreground))",
        },
        secondary: {
          DEFAULT: "hsl(var(--secondary))",
          foreground: "hsl(var(--secondary-foreground))",
        },
        destructive: {
          DEFAULT: "hsl(var(--destructive))",
          foreground: "hsl(var(--destructive-foreground))",
        },
        // Status tokens (DASHBOARD_UI_AUDIT §11): success/warning/info were the
        // missing tier that forced ~641 raw-palette classes. Tint backgrounds
        // and borders via opacity (bg-success/10, border-warning/20) — no
        // separate "subtle" tokens.
        success: {
          DEFAULT: "hsl(var(--success))",
          foreground: "hsl(var(--success-foreground))",
        },
        warning: {
          DEFAULT: "hsl(var(--warning))",
          foreground: "hsl(var(--warning-foreground))",
        },
        info: {
          DEFAULT: "hsl(var(--info))",
          foreground: "hsl(var(--info-foreground))",
        },
        muted: {
          DEFAULT: "hsl(var(--muted))",
          foreground: "hsl(var(--muted-foreground))",
        },
        // Readable-but-secondary text/icon tier (7.25:1) — replaces the
        // failing text-stone-400 habit for meaningful content.
        subtle: "hsl(var(--foreground-subtle))",
        accent: {
          DEFAULT: "hsl(var(--accent))",
          foreground: "hsl(var(--accent-foreground))",
        },
        popover: {
          DEFAULT: "hsl(var(--popover))",
          foreground: "hsl(var(--popover-foreground))",
        },
        card: {
          DEFAULT: "hsl(var(--card))",
          foreground: "hsl(var(--card-foreground))",
        },
        // ---- Tremor tokens (light) — mapped to emerald accent. These are
        // consumed by Tremor's own compiled classes, not app code — keep. ----
        tremor: {
          brand: {
            faint: colors.emerald[50],
            muted: colors.emerald[200],
            subtle: colors.emerald[400],
            DEFAULT: colors.emerald[500],
            emphasis: colors.emerald[700],
            inverted: colors.white,
          },
          background: {
            muted: colors.zinc[50],
            subtle: colors.zinc[100],
            DEFAULT: colors.white,
            emphasis: colors.zinc[700],
          },
          border: { DEFAULT: colors.zinc[200] },
          ring: { DEFAULT: colors.zinc[200] },
          content: {
            subtle: colors.zinc[400],
            DEFAULT: colors.zinc[500],
            emphasis: colors.zinc[700],
            strong: colors.zinc[900],
            inverted: colors.white,
          },
        },
      },
      fontFamily: {
        sans: ["Inter", "ui-sans-serif", "system-ui", "sans-serif"],
        display: ["Inter", "ui-sans-serif", "system-ui", "sans-serif"],
        // `font-mono` utility (IDs, code, tokens) uses the self-hosted JetBrains
        // Mono, matching the .money signature.
        mono: ["JetBrains Mono", "ui-monospace", "SF Mono", "Menlo", "monospace"],
      },
      borderRadius: {
        // Controlled ladder — every step derives from --radius so the whole
        // app moves with one token (rounded-xl/rounded no longer Tailwind
        // defaults that ignore it). Resolved today: 4/4/6/8/12/16px.
        DEFAULT: "calc(var(--radius) - 4px)",
        sm: "calc(var(--radius) - 4px)",
        md: "calc(var(--radius) - 2px)",
        lg: "var(--radius)",
        xl: "calc(var(--radius) + 4px)",
        "2xl": "calc(var(--radius) + 8px)",
        // Tremor
        "tremor-small": "0.375rem",
        "tremor-default": "0.5rem",
        "tremor-full": "9999px",
      },
      fontSize: {
        "tremor-label": ["0.75rem", { lineHeight: "1rem" }],
        "tremor-default": ["0.875rem", { lineHeight: "1.25rem" }],
        "tremor-title": ["1.125rem", { lineHeight: "1.75rem" }],
        "tremor-metric": ["1.875rem", { lineHeight: "2.25rem" }],
      },
      boxShadow: {
        // Elevation ladder (DASHBOARD_UI_AUDIT §11): three levels only.
        // raised   — cards, inputs, buttons (content sitting on the canvas)
        // overlay  — dialogs, sheets, toasts (blocking surfaces)
        // popover  — menus, selects, tooltips, palettes (anchored floats)
        raised: "0 1px 2px 0 rgb(0 0 0 / 0.05)",
        overlay: "0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1)",
        popover: "0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)",
        // Tremor aliases (consumed by Tremor's compiled classes — keep).
        "tremor-input": "0 1px 2px 0 rgb(0 0 0 / 0.05)",
        "tremor-card": "0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1)",
        "tremor-dropdown": "0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)",
      },
      // Motion tokens — mirror of the CSS custom properties in index.css
      // (see frontend/MOTION.md). Use duration-fast/normal/slow +
      // ease-standard/ease-out-soft instead of raw values.
      transitionDuration: {
        fast: "140ms",
        normal: "200ms",
        slow: "340ms",
      },
      transitionTimingFunction: {
        standard: "cubic-bezier(0.2, 0, 0, 1)",
        "out-soft": "cubic-bezier(0.16, 1, 0.3, 1)",
      },
      keyframes: {
        "accordion-down": {
          from: { height: "0" },
          to: { height: "var(--radix-accordion-content-height)" },
        },
        "accordion-up": {
          from: { height: "var(--radix-accordion-content-height)" },
          to: { height: "0" },
        },
        // Mount reveal: fade + a small rise. transform/opacity only.
        "motion-reveal": {
          from: { opacity: "0", transform: "translateY(4px)" },
          to: { opacity: "1", transform: "translateY(0)" },
        },
        // One-shot highlight when a value/status actually changes.
        "motion-flash": {
          from: { backgroundColor: "hsl(var(--primary) / 0.10)" },
          to: { backgroundColor: "transparent" },
        },
      },
      animation: {
        "accordion-down": "accordion-down 0.2s ease-out",
        "accordion-up": "accordion-up 0.2s ease-out",
        "motion-reveal":
          "motion-reveal 200ms cubic-bezier(0.16, 1, 0.3, 1) both",
        "motion-flash": "motion-flash 1200ms ease-out",
      },
    },
  },
  // Safelist Tremor's dynamic color classes so chart/badge colors survive purge.
  safelist: [
    {
      pattern:
        /^(bg|text|border|ring|stroke|fill)-(emerald|zinc|red|amber|blue|violet)-(50|100|200|300|400|500|600|700|800|900)$/,
      variants: ["hover", "ui-selected"],
    },
  ],
  plugins: [require("tailwindcss-animate"), require("@headlessui/tailwindcss")],
};
