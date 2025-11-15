import { createSignal } from 'solid-js';

interface AuthProps {
  onUserIdCreated?: (id: number) => void;
}

export default (props?: AuthProps) => {
  const [email, setEmail] = createSignal<string>('');
  const [name, setName] = createSignal<string>('');
  const [loading, setLoading] = createSignal<boolean>(false);
  const [message, setMessage] = createSignal<string>('');
  const [userId, setUserId] = createSignal<number | null>(null);

  const handleRegister = async () => {
    const emailVal = email().trim();
    const nameVal = name().trim();
    
    if (!emailVal || !nameVal) {
      setMessage('请填写邮箱和姓名');
      return;
    }

    setLoading(true);
    setMessage('');
    setUserId(null);

    try {
      const response = await fetch('/api/v1/users', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          email: emailVal,
          name: nameVal,
        }),
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(JSON.stringify(error));
      }

      const data = await response.json();
      setMessage('注册成功！');
      if (data.id) {
        setUserId(data.id);
        props?.onUserIdCreated?.(data.id);
      }
    } catch (err) {
      setMessage(err instanceof Error ? err.message : '注册失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style="padding: 20px;">
      <h2>用户注册（调试用）</h2>
      <div style="margin-top: 20px;">
        <div style="margin-bottom: 15px;">
          <label style="display: block; margin-bottom: 5px;">邮箱:</label>
          <input
            type="email"
            value={email()}
            onInput={(e) => setEmail(e.currentTarget.value)}
            placeholder="输入邮箱"
            style="padding: 8px; width: 100%; max-width: 300px;"
          />
        </div>
        <div style="margin-bottom: 15px;">
          <label style="display: block; margin-bottom: 5px;">姓名:</label>
          <input
            type="text"
            value={name()}
            onInput={(e) => setName(e.currentTarget.value)}
            placeholder="输入姓名"
            style="padding: 8px; width: 100%; max-width: 300px;"
          />
        </div>
        <button
          onClick={handleRegister}
          disabled={loading()}
          style="padding: 10px 20px; cursor: pointer; background: #667eea; color: white; border: none; border-radius: 4px;"
        >
          {loading() ? '注册中...' : '注册'}
        </button>
      </div>

      {message() && (
        <div style={`margin-top: 15px; padding: 10px; background: ${message().includes('成功') ? '#d4edda' : '#f8d7da'}; border-radius: 4px; color: ${message().includes('成功') ? '#155724' : '#721c24'};`}>
          {message()}
        </div>
      )}

      {userId() && (
        <div style="margin-top: 15px; padding: 10px; background: #d1ecf1; border-radius: 4px; color: #0c5460;">
          用户ID: {userId()}
        </div>
      )}
    </div>
  );
};

