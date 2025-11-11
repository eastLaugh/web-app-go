import type { Component } from 'solid-js';
import Comp from './Comp';
import './App.css';

const App: Component = () => {
  return (
    <div class="app-container">
      <h1>Hello world!!!!</h1>
      <Comp />
    </div>
  );
};

export default App;
