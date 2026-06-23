const fs = require('fs');
const path = require('path');

// Load environment variables from .env file if available
// We don't need dotenv dependency if we assume the environment is already set up
// or if we are running in a context where .env is loaded.
// However, for local dev, it's good to try to read .env manually if dotenv isn't installed
// to avoid adding dependencies if possible. But standard practice is using dotenv.
// Let's check if we can just read the file manually to keep it simple and dependency-free for now,
// or just rely on process.env....

// Simple .env parser
function loadEnv(filePath) {
    if (fs.existsSync(filePath)) {
        const content = fs.readFileSync(filePath, 'utf8');
        content.split('\n').forEach(line => {
            const match = line.match(/^([^=]+)=(.*)$/);
            if (match) {
                const key = match[1].trim();
                const value = match[2].trim().replace(/^["'](.*)["']$/, '$1');
                if (!process.env[key]) {
                    process.env[key] = value;
                }
            }
        });
    }
}

// Try to load from .env in the project root (web/vyst-ui) and potentially parent root
loadEnv(path.join(__dirname, '.env'));
loadEnv(path.join(__dirname, '../../.env')); // Check repository root too

const config = {
    apiUrl: process.env.API_URL || ''
};

const configPath = path.join(__dirname, 'public', 'config.json');
const configDir = path.dirname(configPath);

if (!fs.existsSync(configDir)) {
    fs.mkdirSync(configDir, { recursive: true });
}

fs.writeFileSync(configPath, JSON.stringify(config, null, 2));
console.log(`Generated public/config.json with apiUrl: ${config.apiUrl}`);
