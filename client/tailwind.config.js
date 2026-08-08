/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        obsidian: {
          950: '#0B0B0E',
          900: '#121217',
          800: '#1A1A23',
          700: '#252532',
        },
        gold: {
          50: '#FFFDF0',
          100: '#FFF9C2',
          200: '#FFF085',
          300: '#FFE047',
          400: '#F5CE26',
          500: '#D4AF37', // Metallic Gold Primary
          600: '#AA771C', // Dark Amber Gold
          700: '#855910',
          800: '#643F0E',
          900: '#482D0D',
        },
      },
      fontFamily: {
        sans: ['Inter', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
      boxShadow: {
        'gold-glow': '0 0 25px -5px rgba(212, 175, 55, 0.3)',
        'gold-sm': '0 0 10px rgba(212, 175, 55, 0.2)',
      },
    },
  },
  plugins: [],
};
