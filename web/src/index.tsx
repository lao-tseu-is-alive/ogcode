/* @refresh reload */
import { render } from 'solid-js/web';
import App from './app';
import './styles/index.css';

const root = document.getElementById('root');
if (root) {
  render(() => <App />, root);
}