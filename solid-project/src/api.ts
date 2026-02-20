const API_BASE = '/api/v1';

export const login = async (email: string, code: string = '123456') => {
  const res = await fetch(`${API_BASE}/auth`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, code }),
  });
  if (!res.ok) throw new Error('登录失败');
  const data = await res.json();
  return data.token;
};

export const createPost = async (token: string, file: string, content: string) => {
  const res = await fetch(`${API_BASE}/posts`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ file, content }),
  });
  if (!res.ok) throw new Error('发送失败');
};

export const getPosts = async (file: string) => {
  const res = await fetch(`${API_BASE}/posts?file=${encodeURIComponent(file)}`);
  if (!res.ok) return [];
  return await res.json();
};