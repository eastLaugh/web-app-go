import type { Component } from 'solid-js';
import { createSignal } from 'solid-js';
import { getNextCard } from '../api';

export const AnkiStub: Component = () => {
  const [loading, setLoading] = createSignal(false);

  const handleGo = async () => {
    const token = localStorage.getItem('token');
    if (!token) {
      alert('请先登录');
      return;
    }
    setLoading(true);
    try {
      const nextFile = await getNextCard(token);
      if (!nextFile) {
        alert('没有更多卡片了');
        return;
      }
      window.location.hash = nextFile;
    } catch (e) {
      alert('获取卡片失败');
    } finally {
      setLoading(false);
    }
  };
  

  return (
    <div class="post">
      <div class="post-nav">
        <a onClick={() => (window.location.hash = '')} class="cursor-pointer">
          ← 返回
        </a>
      </div>
      <h2 class="text-lg font-medium mb-2">Anki</h2>
      <div class="text-sm text-gray-600 mb-6">
        点击 GO! 开始复习下一张卡片
      </div>

      <button
        onClick={handleGo}
        disabled={loading()}
        class="px-8 py-3 border border-black bg-black text-white text-base cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {loading() ? '加载中...' : 'GO!'}
      </button>
    </div>
  );
};


