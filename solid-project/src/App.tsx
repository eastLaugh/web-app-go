import type { Component } from 'solid-js';
import { createSignal, createEffect, onMount } from 'solid-js';
import { marked } from 'marked';
import { login, createPost, getPosts } from './api';
import ChatBox, { collapseChat } from './ChatBox';
import './App.css';

const Good: Component<{ title: string; url: string; description?: string }> = (props) => (
  <div class="post-item">
    <a href={props.url} target="_blank" rel="noopener noreferrer" style="color: #000; text-decoration: none; display: block;">
      <div style="font-weight: 500; margin-bottom: 4px;">{props.title}</div>
      {props.description && <div style="font-size: 13px; color: #666;">{props.description}</div>}
    </a>
  </div>
);

const goods = [
  <Good title="示例链接" url="https://example.com" description="这是一个示例链接" />,
];

const App: Component = () => {
  const hash = window.location.hash.slice(1);
  const [currentPage, setCurrentPage] = createSignal(hash.startsWith('goods') ? 'goods' : 'blog');
  const [currentFile, setCurrentFile] = createSignal(hash.startsWith('goods') ? '' : hash);
  const [htmlContent, setHtmlContent] = createSignal('');
  const [posts, setPosts] = createSignal<{ file: string; title: string; time: string }[]>([]);
  const [token, setToken] = createSignal(localStorage.getItem('token') || '');
  const [email, setEmail] = createSignal('');
  const [showLogin, setShowLogin] = createSignal(false);
  const [comments, setComments] = createSignal<any[]>([]);
  const [commentText, setCommentText] = createSignal('');
  const [clickPos, setClickPos] = createSignal({ x: 0, y: 0 });

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
      setComments([]);
      return;
    }
    const res = await fetch(`/app/${file}`);
    if (res.ok) setHtmlContent(await marked(await res.text()));
    const cmts = await getPosts(file);
    setComments(cmts);
  });

  const handleLogin = async () => {
    if (!email()) return;
    try {
      const t = await login(email());
      setToken(t);
      localStorage.setItem('token', t);
      setShowLogin(false);
    } catch (e) {
      alert('登录失败');
    }
  };

  const handleComment = async () => {
    if (!commentText() || !token()) return;
    try {
      await createPost(token(), currentFile(), commentText());
      setCommentText('');
      const cmts = await getPosts(currentFile());
      setComments(cmts);
    } catch (e) {
      alert('发送失败');
    }
  };

  window.addEventListener('hashchange', () => {
    const hash = window.location.hash.slice(1);
    setCurrentPage(hash.startsWith('goods') ? 'goods' : 'blog');
    setCurrentFile(hash.startsWith('goods') ? '' : hash);
  });

  return (
    <div class="blog">
      <header class="blog-header">
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <div style="display: flex; gap: 20px;">
            <h1 onClick={() => { window.location.hash = ''; setCurrentPage('blog'); }} style="cursor: pointer; opacity: currentPage() === 'blog' ? 1 : 0.5;">
              Blog
            </h1>
            <h1 onClick={() => { window.location.hash = 'goods'; setCurrentPage('goods'); }} style="cursor: pointer; opacity: currentPage() === 'goods' ? 1 : 0.5;">
              Goods
            </h1>
          </div>
          {token() ? (
            <span style="font-size: 12px; cursor: pointer;" onClick={(e) => {
              setToken('');
              localStorage.removeItem('token');
              setClickPos({ x: e.clientX, y: e.clientY });
              setShowLogin(true);
            }}>退出</span>
          ) : (
            <span style="font-size: 12px; cursor: pointer;" onClick={(e) => {
              setClickPos({ x: e.clientX, y: e.clientY });
              setShowLogin(true);
            }}>登录</span>
          )}
        </div>
      </header>
      <main class="blog-content" onClick={(e) => {
        // 点击正文内容时收起聊天
        const target = e.target as HTMLElement;
        if (target.closest('.post-body') || target.closest('.post-list') || target.closest('.post-item')) {
          if (collapseChat) {
            collapseChat();
          }
        }
      }}>
        {currentPage() === 'goods' ? (
          <div class="post-list">
            {goods}
          </div>
        ) : !currentFile() ? (
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
            <div class="comments">
              <div style="border-top: 1px solid #000; margin-top: 40px; padding-top: 20px;">
                <div style="font-size: 14px; margin-bottom: 15px;">回复</div>
                {comments().map((c) => (
                  <div style="padding: 10px 0; border-bottom: 1px solid #eee; font-size: 13px;">
                    <div style="color: #999; margin-bottom: 5px;">{c.email || c.author_email}</div>
                    <div>{c.content}</div>
                  </div>
                ))}
                {token() ? (
                  <div style="margin-top: 20px;">
                    <textarea
                      value={commentText()}
                      onInput={(e) => setCommentText(e.currentTarget.value)}
                      placeholder="输入回复..."
                      style="width: 100%; min-height: 60px; padding: 8px; border: 1px solid #000; font-size: 13px; font-family: inherit; resize: vertical;"
                    />
                    <button
                      onClick={handleComment}
                      style="margin-top: 10px; padding: 6px 20px; border: 1px solid #000; background: #fff; cursor: pointer; font-size: 13px;"
                    >
                      发送
                    </button>
                  </div>
                ) : (
                  <div style="margin-top: 20px; font-size: 12px; color: #999;">
                    <a onClick={(e) => {
                      setClickPos({ x: e.clientX, y: e.clientY });
                      setShowLogin(true);
                    }} style="cursor: pointer; text-decoration: underline;">登录</a>后回复
                  </div>
                )}
              </div>
            </div>
          </article>
        )}
      </main>
      {showLogin() && (
        <div class="login-modal" onClick={() => setShowLogin(false)}>
          <div 
            class="login-box"
            style={`transform-origin: ${clickPos().x}px ${clickPos().y}px;`}
            onClick={(e) => e.stopPropagation()}
          >
            <div style="margin-bottom: 20px; font-size: 16px;">登录</div>
            <input
              type="email"
              value={email()}
              onInput={(e) => setEmail(e.currentTarget.value)}
              placeholder="邮箱"
              style="width: 100%; padding: 8px; border: 1px solid #000; font-size: 13px; margin-bottom: 10px;"
              onKeyPress={(e) => e.key === 'Enter' && handleLogin()}
            />
            <div style="font-size: 11px; color: #999; margin-bottom: 15px;">
              验证码功能暂未启用，任意邮箱可登录
            </div>
            <div style="display: flex; gap: 10px;">
              <button
                onClick={handleLogin}
                style="flex: 1; padding: 8px; border: 1px solid #000; background: #000; color: #fff; cursor: pointer; font-size: 13px;"
              >
                登录
              </button>
              <button
                onClick={() => setShowLogin(false)}
                style="flex: 1; padding: 8px; border: 1px solid #000; background: #fff; cursor: pointer; font-size: 13px;"
              >
                取消
              </button>
            </div>
          </div>
        </div>
      )}
      <div style="margin-top: 40px; text-align: center; font-size: 12px; padding-bottom: 100px;">
        <a href="mailto:east_laugh@qq.com" style="color: #000; text-decoration: none;">east_laugh@qq.com</a>
      </div>
      <ChatBox />
    </div>
  );
};

export default App;