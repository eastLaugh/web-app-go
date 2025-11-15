import type { Component } from 'solid-js';
import { createSignal } from 'solid-js';
import Auth from './Auth';
import Comp from './Comp';
import './App.css';

const App: Component = () => {
  const [userId, setUserId] = createSignal<string>('');

  return (
    <div class="app-container">
      <h1>调试界面</h1>
      <Auth onUserIdCreated={(id) => setUserId(String(id))} />
      <Comp initialUserId={userId()} />
    </div>
  );
};

export default App;
