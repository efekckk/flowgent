/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      fontFamily: {
        mono: ['"IBM Plex Mono"', 'ui-monospace', 'SFMono-Regular', 'Menlo', 'monospace'],
        display: ['Fraunces', 'ui-serif', 'Georgia', 'serif'],
      },
      colors: {
        ink: {
          900: '#070B14',
          800: '#0B1220',
          700: '#111A2E',
          600: '#162038',
          500: '#1E2A47',
          400: '#2A3958',
        },
        paper: {
          50:  '#F5F0E8',
          200: '#D8CFBE',
          400: '#94A3B8',
          600: '#5A6885',
        },
        cyan: {
          DEFAULT: '#7DD3FC',
          dim:     '#38BDF8',
          deep:    '#0EA5E9',
        },
        amber: {
          DEFAULT: '#FBBF24',
          deep:    '#D97706',
        },
        rose: {
          DEFAULT: '#FB7185',
          deep:    '#E11D48',
        },
        moss: {
          DEFAULT: '#86EFAC',
          deep:    '#16A34A',
        },
        nodeTrigger: '#7DD3FC',
        nodeLLM:     '#C4B5FD',
        nodeComm:    '#86EFAC',
        nodeData:    '#FDBA74',
        nodeControl: '#FBBF24',
        nodeCode:    '#94A3B8',
      },
      backgroundImage: {
        'blueprint-grid':
          'linear-gradient(to right, rgba(125,211,252,0.06) 1px, transparent 1px), linear-gradient(to bottom, rgba(125,211,252,0.06) 1px, transparent 1px)',
        'blueprint-grid-fine':
          'linear-gradient(to right, rgba(125,211,252,0.04) 1px, transparent 1px), linear-gradient(to bottom, rgba(125,211,252,0.04) 1px, transparent 1px)',
        'vignette':
          'radial-gradient(ellipse at center, transparent 40%, rgba(0,0,0,0.55) 100%)',
      },
      backgroundSize: {
        'grid-12': '12px 12px',
        'grid-48': '48px 48px',
        'grid-96': '96px 96px',
      },
      borderRadius: {
        sharp: '2px',
      },
      boxShadow: {
        callout: '0 0 0 1px rgba(125,211,252,0.15), 0 12px 32px -16px rgba(0,0,0,0.6)',
        inset:   'inset 0 0 0 1px rgba(125,211,252,0.10)',
      },
      keyframes: {
        'draft-in': {
          '0%':   { opacity: '0', transform: 'translateY(4px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        'tick': {
          '0%, 49%':   { opacity: '1' },
          '50%, 100%': { opacity: '0.2' },
        },
      },
      animation: {
        'draft-in': 'draft-in 380ms cubic-bezier(0.2, 0.7, 0.2, 1) both',
        'tick':     'tick 1.4s steps(2, end) infinite',
      },
    },
  },
  plugins: [],
};
