import type { Component } from 'solid-js';
import { createSignal, createEffect } from 'solid-js';
import { marked } from 'marked';
import { login, createPost, getPosts } from './api';
import ChatBox, { collapseChat } from './ChatBox';
import './App.css';

const Good: Component<{ title: string; url: string; description?: string }> = (props) => (
  <div class="post-item">
    <a href={props.url} target="_blank" rel="noopener noreferrer" class="text-black no-underline block">
      <div class="font-medium mb-1">{props.title}</div>
      {props.description && <div class="text-[13px] text-gray-600">{props.description}</div>}
    </a>
  </div>
);

const Post: Component<{ file: string; title: string; time?: string }> = (props) => {
  const formatTime = (time: string) => {
    const days = Math.floor((Date.now() - new Date(time).getTime()) / 86400000);
    if (days === 0) return '今天';
    if (days === 1) return '昨天';
    if (days < 7) return `${days}天前`;
    if (days < 30) return `${Math.floor(days / 7)}周前`;
    if (days < 365) return `${Math.floor(days / 30)}个月前`;
    return `${Math.floor(days / 365)}年前`;
  };
  return (
    <div class="post-item" onClick={() => (window.location.hash = props.file)}>
      <span>{props.title}</span>
      {props.time && <span class="post-date">{formatTime(props.time)}</span>}
    </div>
  );
};

const App: Component = () => {
  const hash = window.location.hash.slice(1);
  const [currentPage, setCurrentPage] = createSignal(hash.startsWith('goods') ? 'goods' : 'blog');
  const [currentFile, setCurrentFile] = createSignal(hash.startsWith('goods') ? '' : hash);
  const [htmlContent, setHtmlContent] = createSignal('');
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
        <div class="flex justify-between items-center">
          <div class="flex gap-5">
            <h1 onClick={() => { window.location.hash = ''; setCurrentPage('blog'); }} class={`cursor-pointer ${currentPage() === 'blog' ? 'opacity-100' : 'opacity-50'}`}>
              Blog
            </h1>
            <h1 onClick={() => { window.location.hash = 'goods'; setCurrentPage('goods'); }} class={`cursor-pointer ${currentPage() === 'goods' ? 'opacity-100' : 'opacity-50'}`}>
              Goods
            </h1>
          </div>
          {token() ? (
            <span class="text-xs cursor-pointer" onClick={(e) => {
              setToken('');
              localStorage.removeItem('token');
              setClickPos({ x: e.clientX, y: e.clientY });
              setShowLogin(true);
            }}>退出</span>
          ) : (
            <span class="text-xs cursor-pointer" onClick={(e) => {
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
            <Good title="示例链接" url="https://example.com" description="这是一个示例链接" />
            <Good title="Go Green Tea GC" url="https://tonybai.com/2025/10/31/deep-into-go-green-tea-gc/" description="垃圾回收器：从 DFS 到 BFS" />
          </div>
        ) : !currentFile() ? (

          <div class="post-list">
            <Post file="blogs/cong-mysql-dao-mongodb.md" title="从 MySQL 迁移到 MongoDB" time="2025-12-22T01:28:00.000+08:00" />
            <Post file="blogs/rag-shi-xian.md" title="为个人网站添加 RAG 功能" time="2025-12-20T19:30:00.000+08:00" />

            <Post file="blogs/calculate.md" title="计算器" time="2025-12-16T06:44:06.962Z" />
            <Post file="blogs/yi-chu-gin.md" title="移除了 Gin，拥抱标准库" time="2025-12-12T20:30:39.589Z" />
            <Post file="blogs/1213test.md" title="1213test" time="2025-12-12T19:57:27.981Z" />
            <Post file="blogs/picture_test.md" title="测试图片渲染" time="2025-12-08T16:23:48.082Z" />
            <Post file="blogs/zhuang-tai-ji.md" title="状态机" time="2025-11-26T07:11:32.852Z" />
            <Post file="blogs/shan-dang-ce-shi.md" title="删档测试" time="2025-11-25T10:05:30.863Z" />
            <Post file="blogs/post.md" title="Hello World" time="2025-11-22T09:38:12.950Z" />
          </div>
        ) : (
          <article class="post">
            <div class="post-nav">
              <a onClick={() => (window.location.hash = '')} class="cursor-pointer">
                ← 返回
              </a>
            </div>
            <div class="post-body" innerHTML={htmlContent()} />
            <div class="comments">
              <div class="border-t border-black mt-10 pt-5">
                <div class="text-sm mb-4">回复</div>
                {comments().map((c) => (
                  <div class="py-2.5 border-b border-gray-200 text-[13px]">
                    <div class="text-gray-500 mb-1.5">{c.email || c.author_email}</div>
                    <div>{c.content}</div>
                  </div>
                ))}
                {token() ? (
                  <div class="mt-5">
                    <textarea
                      value={commentText()}
                      onInput={(e) => setCommentText(e.currentTarget.value)}
                      placeholder="输入回复..."
                      class="w-full min-h-[60px] p-2 border border-black text-[13px] font-inherit resize-y"
                    />
                    <button
                      onClick={handleComment}
                      class="mt-2.5 px-5 py-1.5 border border-black bg-white cursor-pointer text-[13px]"
                    >
                      发送
                    </button>
                  </div>
                ) : (
                  <div class="mt-5 text-xs text-gray-500">
                    <a onClick={(e) => {
                      setClickPos({ x: e.clientX, y: e.clientY });
                      setShowLogin(true);
                    }} class="cursor-pointer underline">登录</a>后回复
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
            <div class="mb-5 text-base">登录</div>
            <input
              type="email"
              value={email()}
              onInput={(e) => setEmail(e.currentTarget.value)}
              placeholder="邮箱"
              class="w-full p-2 border border-black text-[13px] mb-2.5"
              onKeyPress={(e) => e.key === 'Enter' && handleLogin()}
            />
            <div class="text-[11px] text-gray-500 mb-4">
              验证码功能暂未启用，任意邮箱可登录
            </div>
            <div class="flex gap-2.5">
              <button
                onClick={handleLogin}
                class="flex-1 p-2 border border-black bg-black text-white cursor-pointer text-[13px]"
              >
                登录
              </button>
              <button
                onClick={() => setShowLogin(false)}
                class="flex-1 p-2 border border-black bg-white cursor-pointer text-[13px]"
              >
                取消
              </button>
            </div>
          </div>
        </div>
      )}
      <div class="mt-10 text-center text-xs pb-[100px]">
        <a href="mailto:east_laugh@qq.com" class="text-black no-underline">east_laugh@qq.com</a>
      </div>
      <ChatBox />
    </div>
  );
};

export default App;