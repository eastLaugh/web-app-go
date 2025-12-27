import type { Component } from 'solid-js';
import { createSignal, For, createEffect } from 'solid-js';
import './ChatBox.css';

interface Message {
  role: 'user' | 'assistant';
  content: string;
}

const STORAGE_KEY = 'chat_messages';
const COLLAPSED_KEY = 'chat_collapsed';

// 全局收起函数
let collapseChatFn: (() => void) | undefined;

export const collapseChat = () => {
  if (collapseChatFn) {
    collapseChatFn();
  }
};

const ChatBox: Component = () => {
  // 从 localStorage 加载消息
  const loadMessages = (): Message[] => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored) {
        return JSON.parse(stored);
      }
    } catch (e) {
      console.error('加载聊天记录失败:', e);
    }
    return [];
  };

  const [messages, setMessages] = createSignal<Message[]>(loadMessages());
  const [input, setInput] = createSignal('');
  const [isLoading, setIsLoading] = createSignal(false);
  const [currentContent, setCurrentContent] = createSignal('');
  const [isCollapsed, setIsCollapsed] = createSignal((() => {
    try {
      return localStorage.getItem(COLLAPSED_KEY) === 'true';
    } catch {
      return false;
    }
  })());
  let messagesContentContainer: HTMLDivElement | undefined;
  let inputContainer: HTMLDivElement | undefined;
  let wrapperElement: HTMLDivElement | undefined;

  // 暴露收起函数
  collapseChatFn = () => {
    setIsCollapsed(true);
    try {
      localStorage.setItem(COLLAPSED_KEY, 'true');
    } catch (e) {
      console.error('保存收起状态失败:', e);
    }
  };

  // 保存消息到 localStorage
  createEffect(() => {
    const msgs = messages();
    try {
      if (msgs.length > 0) {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(msgs));
      } else {
        localStorage.removeItem(STORAGE_KEY);
      }
    } catch (e) {
      console.error('保存聊天记录失败:', e);
    }
  });

  // 自动滚动到底部
  createEffect(() => {
    const msgs = messages();
    if (messagesContentContainer && msgs.length > 0) {
      // 使用 setTimeout 确保 DOM 更新完成
      setTimeout(() => {
        if (messagesContentContainer) {
          messagesContentContainer.scrollTop = messagesContentContainer.scrollHeight;
        }
      }, 0);
    }
  });

  createEffect(() => {
    if (currentContent() && messagesContentContainer) {
      // 流式输出时实时滚动
      setTimeout(() => {
        if (messagesContentContainer) {
          messagesContentContainer.scrollTop = messagesContentContainer.scrollHeight;
        }
      }, 0);
    }
  });

  const sendMessage = async () => {
    const text = input().trim();
    if (!text || isLoading()) return;

    const userMessage: Message = { role: 'user', content: text };
    const updatedMessages = [...messages(), userMessage];
    setMessages(updatedMessages);
    setInput('');
    setIsLoading(true);
    setCurrentContent('');

    try {
      const response = await fetch('/api/v1/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          messages: updatedMessages.map((m) => ({
            role: m.role,
            content: m.content,
          })),
        }),
      });

      if (!response.ok) {
        throw new Error('请求失败');
      }

      const reader = response.body?.getReader();
      const decoder = new TextDecoder();

      if (!reader) {
        throw new Error('无法读取响应流');
      }

      let buffer = '';
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            const text = line.slice(6);
            if (text.startsWith('[ERROR]')) {
              throw new Error(text.slice(7));
            }
            if (text) {
              setCurrentContent(prev => prev + text);
            }
          }
        }
      }

      const finalContent = currentContent();
      if (finalContent) {
        setMessages(prev => [...prev, { role: 'assistant', content: finalContent }]);
        setCurrentContent('');
      }
      setIsLoading(false);
    } catch (error) {
      console.error('发送消息失败:', error);
      setMessages(prev => [...prev, { role: 'assistant', content: '抱歉，发送消息失败，请稍后重试。' }]);
      setIsLoading(false);
      setCurrentContent('');
    }
  };

  const handleKeyPress = (e: KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      const text = input().trim();
      // 检查是否是 /clear 命令
      if (text === '/clear') {
        setMessages([]);
        setInput('');
        try {
          localStorage.removeItem(STORAGE_KEY);
        } catch (e) {
          console.error('清除聊天记录失败:', e);
        }
        return;
      }
      sendMessage();
    }
  };

  const handleCollapse = () => {
    setIsCollapsed(true);
    try {
      localStorage.setItem(COLLAPSED_KEY, 'true');
    } catch (e) {
      console.error('保存收起状态失败:', e);
    }
  };

  const handleExpand = () => {
    setIsCollapsed(false);
    try {
      localStorage.removeItem(COLLAPSED_KEY);
    } catch (e) {
      console.error('清除收起状态失败:', e);
    }
  };

  // 点击输入框展开
  const handleInputClick = () => {
    if (isCollapsed()) {
      handleExpand();
    }
  };

  return (
    <div class="chat-wrapper" ref={wrapperElement}>
      {messages().length > 0 || isLoading() ? (
        <div 
          class={`chat-messages ${isCollapsed() ? 'collapsed' : ''}`}
        >
          <button class="chat-collapse-btn" onClick={handleCollapse} title="收起">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
              <path d="M8 4L4 8L8 12" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" transform="rotate(-90 8 8)"/>
            </svg>
          </button>
          <div class="chat-messages-content" ref={messagesContentContainer}>
            <For each={messages()}>
              {(msg) => (
                <div class={`chat-message ${msg.role}`}>
                  <div class="chat-message-content">{msg.content}</div>
                </div>
              )}
            </For>
            {isLoading() && currentContent() && (
              <div class="chat-message assistant">
                <div class="chat-message-content">{currentContent()}</div>
              </div>
            )}
          </div>
        </div>
      ) : null}
      <div class="chat-input-container" ref={inputContainer} onClick={handleInputClick}>
        <textarea
          class="chat-input"
          value={input()}
          onInput={(e) => setInput(e.currentTarget.value)}
          onKeyPress={handleKeyPress}
          placeholder={isCollapsed() ? '点击输入框展开聊天' : '按 Enter 与"我"聊天'}
          disabled={isLoading()}
          rows={1}
        />
      </div>
    </div>
  );
};

export default ChatBox;

