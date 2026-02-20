import type { Component } from 'solid-js';
import { createSignal, For, createEffect } from 'solid-js';
import './ChatBox.css';

interface Message {
  role: 'user' | 'assistant';
  content: string;
  toolCalls?: string[];
}

const COLLAPSED_KEY = 'chat_collapsed';

// 全局收起函数
let collapseChatFn: (() => void) | undefined;

export const collapseChat = () => {
  if (collapseChatFn) {
    collapseChatFn();
  }
};

function apiMessagesToMessages(api: Array<Record<string, unknown>>): Message[] {
  const out: Message[] = [];
  for (const m of api) {
    const role = m.role as string;
    if (role === 'system') continue;
    const content = (m.content as string) ?? '';
    const toolCalls = (m.tool_calls as Array<{ function?: { name?: string } }>)?.map((t) => t.function?.name).filter(Boolean) as string[] | undefined;
    out.push({ role: role as 'user' | 'assistant', content, toolCalls: toolCalls?.length ? toolCalls : undefined });
  }
  return out;
}

const ChatBox: Component = () => {
  const [conversationId, setConversationId] = createSignal<string | null>(null);
  const [conversationList, setConversationList] = createSignal<string[]>([]);
  const [conversationTitles, setConversationTitles] = createSignal<Record<string, string>>({});
  const [messages, setMessages] = createSignal<Message[]>([]);
  const [input, setInput] = createSignal('');
  const [isLoading, setIsLoading] = createSignal(false);
  const [currentContent, setCurrentContent] = createSignal('');
  const [callingTools, setCallingTools] = createSignal<string[] | null>(null);
  const [toolsCalledInReply, setToolsCalledInReply] = createSignal<string[]>([]);
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
    currentContent();
    callingTools();
    toolsCalledInReply();
    if (messagesContentContainer) {
      setTimeout(() => {
        if (messagesContentContainer) {
          messagesContentContainer.scrollTop = messagesContentContainer.scrollHeight;
        }
      }, 0);
    }
  });

  const handleNewConversation = () => {
    setConversationId(null);
    setMessages([]);
  };

  const authHeaders = (): Record<string, string> => {
    const t = typeof localStorage !== 'undefined' ? localStorage.getItem('token') : null;
    return t ? { Authorization: `Bearer ${t}` } : {};
  };

  const fetchConversationList = async () => {
    const res = await fetch('/api/v1/conversations', { headers: authHeaders() });
    if (!res.ok) return;
    const data = await res.json();
    setConversationList(Array.isArray(data.ids) ? data.ids : []);
    setConversationTitles(data.titles && typeof data.titles === 'object' ? data.titles : {});
  };

  createEffect(() => {
    if (typeof localStorage !== 'undefined' && localStorage.getItem('token')) {
      fetchConversationList();
    }
  });

  const openConversation = async (id: string) => {
    const res = await fetch(`/api/v1/conversations/${encodeURIComponent(id)}`, { headers: authHeaders() });
    if (!res.ok) return;
    const apiMessages = (await res.json()) as Array<Record<string, unknown>>;
    setConversationId(id);
    setMessages(apiMessagesToMessages(apiMessages));
  };

  const ensureConversation = async (): Promise<string> => {
    const cur = conversationId();
    if (cur) return cur;
    const res = await fetch('/api/v1/conversations', { method: 'POST', headers: authHeaders() });
    if (!res.ok) throw new Error('创建对话失败');
    const data = await res.json();
    const id = data.id as string;
    setConversationId(id);
    setConversationList((prev) => [id, ...prev]);
    fetchConversationList();
    return id;
  };

  const sendMessage = async () => {
    const text = input().trim();
    if (!text || isLoading()) return;

    const userMessage: Message = { role: 'user', content: text };
    setMessages((prev) => [...prev, userMessage]);
    setInput('');
    setIsLoading(true);
    setCurrentContent('');
    setCallingTools(null);
    setToolsCalledInReply([]);

    try {
      const convId = await ensureConversation();
      const response = await fetch('/api/v1/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeaders() },
        body: JSON.stringify({ conversation_id: convId, content: text }),
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
              try {
                const parsed = JSON.parse(text);
                if (parsed && parsed.event === 'tool_call' && Array.isArray(parsed.tools)) {
                  setCallingTools(parsed.tools);
                  setToolsCalledInReply(prev => [...prev, ...parsed.tools]);
                } else if (typeof parsed === 'string') {
                  setCallingTools(null);
                  setCurrentContent(prev => prev + parsed);
                } else {
                  setCurrentContent(prev => prev + String(parsed));
                }
              } catch {
                setCallingTools(null);
                setCurrentContent(prev => prev + text);
              }
            }
          }
        }
      }

      const finalContent = currentContent();
      const toolsCalled = toolsCalledInReply();
      setCallingTools(null);
      setToolsCalledInReply([]);
      if (finalContent || toolsCalled.length > 0) {
        setMessages(prev => [...prev, { role: 'assistant', content: finalContent, toolCalls: toolsCalled.length > 0 ? toolsCalled : undefined }]);
        setCurrentContent('');
      }
      setIsLoading(false);
      fetchConversationList();
    } catch (error) {
      console.error('发送消息失败:', error);
      setMessages(prev => [...prev, { role: 'assistant', content: '抱歉，发送消息失败，请稍后重试。' }]);
      setIsLoading(false);
      setCurrentContent('');
      setCallingTools(null);
      setToolsCalledInReply([]);
    }
  };

  const handleKeyPress = (e: KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
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
      {conversationId() != null || messages().length > 0 || isLoading() ? (
        <div 
          class={`chat-messages ${isCollapsed() ? 'collapsed' : ''}`}
        >
          {conversationId() != null && (
            <div class="chat-session-id" title={conversationId()!}>
              {conversationId()}
            </div>
          )}
          <button class="chat-collapse-btn" onClick={handleCollapse} title="收起">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
              <path d="M8 4L4 8L8 12" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" transform="rotate(-90 8 8)"/>
            </svg>
          </button>
          <div class="chat-messages-content" ref={messagesContentContainer}>
            <For each={messages()}>
              {(msg) => (
                <div class={`chat-message ${msg.role}`}>
                  {msg.content ? <div class="chat-message-content">{msg.content}</div> : null}
                  {msg.role === 'assistant' && msg.toolCalls?.map((name) => (
                    <div class="chat-tool-done">调用了 {name}</div>
                  ))}
                </div>
              )}
            </For>
            {isLoading() && (currentContent() || toolsCalledInReply().length > 0 || (callingTools() && callingTools()!.length > 0)) && (
              <div class="chat-message assistant chat-message-inprogress">
                {currentContent() && <div class="chat-message-content">{currentContent()}</div>}
                <For each={toolsCalledInReply()}>
                  {(name) => <div class="chat-tool-done">调用了 {name}</div>}
                </For>
                {callingTools() && callingTools()!.length > 0 && (
                  <div class="chat-tool-calling">
                    正在调用：{callingTools()!.join('、')}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      ) : null}
      <div class="chat-input-container" ref={inputContainer} onClick={handleInputClick}>
        <select
          class="chat-history-select"
          value={conversationId() ?? ''}
          onChange={(e) => {
            const id = e.currentTarget.value;
            if (id) openConversation(id);
            else handleNewConversation();
          }}
          onClick={(e) => e.stopPropagation()}
          title="新建会话"
        >
          <option value="">历史会话</option>
            <For each={conversationList()}>
              {(id) => <option value={id}>{conversationTitles()[id] || `${id.slice(0, 8)}…`}</option>}
            </For>
        </select>
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

