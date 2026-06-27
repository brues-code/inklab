/** @type {import('tailwindcss').Config} */
export default {
    content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
    theme: {
        extend: {
            colors: {
                wow: {
                    poor: '#9d9d9d',
                    common: '#ffffff',
                    uncommon: '#1eff00',
                    rare: '#0070dd',
                    epic: '#a335ee',
                    legendary: '#ff8000',
                    artifact: '#e6cc80',
                    heirloom: '#ff0000',
                    gold: '#ffd100',
                    alliance: '#0070DE',
                    horde: '#C41F3B',
                },
                bg: {
                    dark: '#0a0a0a',
                    main: '#181818',
                    panel: '#242424',
                    hover: '#303030',
                    active: '#404040',
                },
                border: {
                    dark: '#282828',
                    light: '#404040',
                    highlight: '#505050',
                },
                money: {
                    gold: '#ffd700',
                    silver: '#c0c0c0',
                    copper: '#b87333',
                },
            },
            fontFamily: {
                wow: ['"Friz Quadrata"', 'Georgia', 'serif'],
            },
        },
    },
    plugins: [],
}
