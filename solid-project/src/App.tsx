import type { Component } from 'solid-js';
import { createSignal, createEffect, onMount } from 'solid-js';
import { marked } from 'marked';
import './App.css';

const App: Component = () => {
  const [currentFile, setCurrentFile] = createSignal(window.location.hash.slice(1));
  const [htmlContent, setHtmlContent] = createSignal('');
  const [posts, setPosts] = createSignal<{ file: string; title: string; time: string }[]>([]);

  const formatTime = (time: string) => {
    const days = Math.floor((Date.now() - new Date(time).getTime()) / 86400000);
    if (days === 0) return '今天';
    if (days === 1) return '昨天';
    if (days < 7) return `${days}天前`;
    if (days < 30) return `${Math.floor(days / 7)}周前`;
    if (days < 365) return `${Math.floor(days / 30)}个月前`;
    return `${Math.floor(days / 365)}年前`;
  };

  onMount(async () => {
    const res = await fetch('/app/posts.json');
    if (res.ok) setPosts(await res.json());
  });

  createEffect(async () => {
    const file = currentFile();
    if (!file) {
      setHtmlContent('');
      return;
    }
    const res = await fetch(`/app/${file}`);
    if (res.ok) setHtmlContent(await marked(await res.text()));
  });

  window.addEventListener('hashchange', () => setCurrentFile(window.location.hash.slice(1)));

  return (
    <div class="blog">
      <header class="blog-header">
        <h1 onClick={() => (window.location.hash = '')} style="cursor: pointer;">
          Blog
        </h1>
      </header>
      <main class="blog-content">
        {!currentFile() ? (
          <div class="post-list">
            {posts().map((p) => (
              <div class="post-item" onClick={() => (window.location.hash = p.file)}>
                <span>{p.title}</span>
                {p.time && <span class="post-date">{formatTime(p.time)}</span>}
              </div>
            ))}
          </div>
        ) : (
          <article class="post">
            <div class="post-nav">
              <a onClick={() => (window.location.hash = '')} style="cursor: pointer;">
                ← 返回
              </a>
              {(() => {
                const p = posts().find((x) => x.file === currentFile());
                return p?.time && <span class="post-date">{formatTime(p.time)}</span>;
              })()}
            </div>
            <div class="post-body" innerHTML={htmlContent()} />
          </article>
        )}
      </main>
      <div style="margin-top: 40px; text-align: center; font-size: 12px;">
        <a href="mailto:east_laugh@qq.com" style="color: #000; text-decoration: none;">east_laugh@qq.com</a>
      </div>
    </div>
  );
};

export default App;