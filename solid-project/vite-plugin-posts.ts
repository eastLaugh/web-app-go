import { Plugin } from 'vite';
import { readdirSync, statSync, writeFileSync, existsSync, mkdirSync, readFileSync } from 'fs';
import { join } from 'path';

const getTitle = (content: string) => {
  const firstLine = content.split('\n')[0].trim();
  return firstLine.replace(/^#+\s*/, '') || 'Untitled';
};

const getPosts = () => {
  const publicDir = join(process.cwd(), 'public');
  const files = readdirSync(publicDir).filter((f) => f.endsWith('.md'));
  return files.map((file) => {
    const content = readFileSync(join(publicDir, file), 'utf-8');
    return {
      file,
      title: getTitle(content),
      time: statSync(join(publicDir, file)).mtime.toISOString(),
    };
  });
};

export default function postsPlugin(): Plugin {
  return {
    name: 'posts-timestamps',
    configureServer(server) {
      server.middlewares.use('/app/posts.json', (req, res) => {
        res.setHeader('Content-Type', 'application/json');
        res.end(JSON.stringify(getPosts()));
      });
    },
    closeBundle() {
      const distDir = join(process.cwd(), '../go/cmd/server/dist');
      if (!existsSync(distDir)) mkdirSync(distDir, { recursive: true });
      writeFileSync(join(distDir, 'posts.json'), JSON.stringify(getPosts()));
    },
  };
}
