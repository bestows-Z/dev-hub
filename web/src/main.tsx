import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import App from './App'
import { FeedbackProvider } from './Feedback'
import './styles.css'
import './blog-theme.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <BrowserRouter>
      <FeedbackProvider><App /></FeedbackProvider>
    </BrowserRouter>
  </React.StrictMode>,
)
