import type { Config } from "tailwindcss";

// Design system tokens from Stitch "Industrial Sentinel" spec
const config: Config = {
  darkMode: "class",
  content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        // Surface hierarchy (Level 0-3)
        "surface-container-lowest": "#060e20",
        "surface-container-low": "#131b2e",
        "surface-container": "#171f33",
        "surface-container-high": "#222a3d",
        "surface-container-highest": "#2d3449",
        "surface-dim": "#0b1326",
        "surface-bright": "#31394d",
        "surface-variant": "#2d3449",
        surface: "#0b1326",
        background: "#0b1326",
        "on-background": "#dae2fd",

        // Primary (action)
        primary: "#adc6ff",
        "primary-container": "#4d8eff",
        "primary-fixed": "#d8e2ff",
        "primary-fixed-dim": "#adc6ff",
        "on-primary": "#002e6a",
        "on-primary-container": "#00285d",
        "on-primary-fixed": "#001a42",
        "on-primary-fixed-variant": "#004395",
        "inverse-primary": "#005ac2",

        // Secondary
        secondary: "#bcc7de",
        "secondary-container": "#3e495d",
        "secondary-fixed": "#d8e3fb",
        "secondary-fixed-dim": "#bcc7de",
        "on-secondary": "#263143",
        "on-secondary-container": "#aeb9d0",
        "on-secondary-fixed": "#111c2d",
        "on-secondary-fixed-variant": "#3c475a",

        // Tertiary (healthy/online)
        tertiary: "#4edea3",
        "tertiary-container": "#00a572",
        "tertiary-fixed": "#6ffbbe",
        "tertiary-fixed-dim": "#4edea3",
        "on-tertiary": "#003824",
        "on-tertiary-container": "#00311f",
        "on-tertiary-fixed": "#002113",
        "on-tertiary-fixed-variant": "#005236",

        // Error / critical
        error: "#ffb4ab",
        "error-container": "#93000a",
        "on-error": "#690005",
        "on-error-container": "#ffdad6",

        // Neutral / surfaces
        "on-surface": "#dae2fd",
        "on-surface-variant": "#c2c6d6",
        outline: "#8c909f",
        "outline-variant": "#424754",
        "inverse-surface": "#dae2fd",
        "inverse-on-surface": "#283044",
        "surface-tint": "#adc6ff",
      },
      fontFamily: {
        headline: ["Manrope", "sans-serif"],
        body: ["Inter", "sans-serif"],
        label: ["Inter", "sans-serif"],
        technical: ["JetBrains Mono", "monospace"],
      },
      borderRadius: {
        DEFAULT: "0.25rem",
        lg: "0.5rem",
        xl: "1.5rem",
        full: "9999px",
      },
      animation: {
        "pulse-glow": "pulse-glow 2s infinite ease-in-out",
        "pulse-soft": "pulse-soft 2s infinite ease-in-out",
        "fade-in": "fade-in 0.2s ease-out",
        "slide-in-left": "slide-in-left 0.3s ease-out",
      },
      keyframes: {
        "pulse-glow": {
          "0%": {
            transform: "scale(0.8)",
            opacity: "0.5",
            boxShadow: "0 0 0 0 rgba(78, 222, 163, 0.7)",
          },
          "70%": {
            transform: "scale(1.2)",
            opacity: "1",
            boxShadow: "0 0 0 8px rgba(78, 222, 163, 0)",
          },
          "100%": {
            transform: "scale(0.8)",
            opacity: "0.5",
            boxShadow: "0 0 0 0 rgba(78, 222, 163, 0)",
          },
        },
        "pulse-soft": {
          "0%": { opacity: "0.4", transform: "scale(0.95)" },
          "50%": { opacity: "1", transform: "scale(1.05)" },
          "100%": { opacity: "0.4", transform: "scale(0.95)" },
        },
        "fade-in": {
          from: { opacity: "0", transform: "translateY(4px)" },
          to: { opacity: "1", transform: "translateY(0)" },
        },
        "slide-in-left": {
          from: { transform: "translateX(-100%)" },
          to: { transform: "translateX(0)" },
        },
      },
      boxShadow: {
        ambient:
          "0 0 32px rgba(0, 26, 66, 0.08), 0 4px 16px rgba(0, 26, 66, 0.04)",
        card: "0 4px 24px rgba(0, 26, 66, 0.12)",
        primary: "0 0 20px rgba(173, 198, 255, 0.4)",
      },
    },
  },
  plugins: [],
};

export default config;
