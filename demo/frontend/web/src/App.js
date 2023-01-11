import React from 'react'
import { HashRouter, Routes, Route } from 'react-router-dom'
import { DemoMenu } from './components/DemoMenu.jsx'
import { DemoOverview } from './components/DemoOverview.jsx'
import { JsonViewer } from './components/JsonViewer.jsx'
import '@elastic/eui/dist/eui_theme_light.css'

import { EuiProvider, EuiPage } from '@elastic/eui'

const App = () => (
  <HashRouter>
    <EuiProvider colorMode="light">
      <EuiPage>
        <Routes>
          <Route exact path='/' element={ <DemoMenu />} />
          <Route exact path='/directive/:directiveFile?' element={ <JsonViewer />} />
          <Route exact path='/overview' element={ <DemoOverview />} />
        </Routes>
      </EuiPage>
    </EuiProvider>
  </HashRouter>
)

export default App
