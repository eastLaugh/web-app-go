import { createSignal, createEffect } from 'solid-js';

interface CompProps {
  initialUserId?: string;
}

export default (props?: CompProps) => {
  const [userId, setUserId] = createSignal<string>(props?.initialUserId || '1');
  
  createEffect(() => {
    if (props?.initialUserId) {
      setUserId(props.initialUserId);
    }
  });
  const [userInfo, setUserInfo] = createSignal<any>(null);
  const [loading, setLoading] = createSignal<boolean>(false);
  const [error, setError] = createSignal<string>('');

  const fetchUserInfo = async () => {
    const id = userId().trim();
    if (!id) {
      setError('请输入用户 ID');
      return;
    }

    setLoading(true);
    setError('');
    setUserInfo(null);

    try {
      const response = await fetch(`/api/v1/users/${id}`);
      if (!response.ok) {
        const error = await response.json();
        throw new Error(JSON.stringify(error));
      }
      const data = await response.json();
      setUserInfo(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : '获取用户信息失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style="padding: 20px;">
      <h2>调试用户信息</h2>
      <div style="margin-bottom: 15px;">
        <label style="display: block; margin-bottom: 5px;">用户 ID:</label>
        <input
          type="text"
          value={userId()}
          onInput={(e) => setUserId(e.currentTarget.value)}
          placeholder="输入用户 ID"
          style="padding: 8px; width: 200px; margin-right: 10px;"
        />
        <button
          onClick={fetchUserInfo}
          disabled={loading()}
          style="padding: 8px 16px; cursor: pointer;"
        >
          {loading() ? '加载中...' : '获取用户信息'}
        </button>
      </div>

      {error() && (
        <div style="color: red; margin-top: 10px;">
          错误: {error()}
        </div>
      )}

      {userInfo() && (
        <div style="margin-top: 15px; padding: 10px; background: #f5f5f5; border-radius: 4px;">
          <h3>用户信息:</h3>
          <pre style="white-space: pre-wrap; word-wrap: break-word;">
            {JSON.stringify(userInfo(), null, 2)}
          </pre>
        </div>
      )}
    </div>
  );
};
