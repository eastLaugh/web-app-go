import type { Component } from 'solid-js';

export const AnkiStub: Component = () => {
  return (
    <div class="post">
      <div class="post-nav">
        <a onClick={() => (window.location.hash = '')} class="cursor-pointer">
          ← 返回
        </a>
      </div>
      <h2 class="text-lg font-medium mb-2">Anki（Stub）</h2>
      <div class="text-sm text-gray-600 mb-6">
        这是一个精简版 Anki 的占位页面：路由与页面结构已就位，具体功能后续再补。
      </div>

      <div class="border border-black p-4 mb-6">
        <div class="text-sm mb-3">卡组</div>
        <div class="text-[13px] text-gray-700 space-y-2">
          <div class="flex justify-between">
            <span>示例卡组</span>
            <span>新卡 0 · 待复习 0</span>
          </div>
          <div class="flex justify-between opacity-60">
            <span>（占位）英语</span>
            <span>新卡 0 · 待复习 0</span>
          </div>
        </div>
      </div>

      <div class="flex gap-2.5">
        <button class="px-5 py-1.5 border border-black bg-white cursor-not-allowed text-[13px] opacity-60" disabled>
          开始复习（TODO）
        </button>
        <button class="px-5 py-1.5 border border-black bg-white cursor-not-allowed text-[13px] opacity-60" disabled>
          新建卡片（TODO）
        </button>
      </div>
    </div>
  );
};


