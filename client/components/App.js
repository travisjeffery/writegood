import React from 'react'
import DocumentList from '../containers/DocumentList'
import Editor from '../containers/Editor'
import { Frame, TopBar, Page, Card, Layout } from '@shopify/polaris'

const App = () => (
  <Page
    title='My first document'
    primaryAction={{content: 'New'}}
    secondaryActions={[
      {
        content: 'Save Draft',
      },
      {
        content: 'Rename',
      },
      {
        content: 'Delete',
      }
    ]}
  >
    <Layout>
      <Layout.Section>
        <Card sectioned>
          <Editor />
        </Card>
      </Layout.Section>
      <Layout.Section secondary>
        <Card title="Drafts" sectioned>
          <p>Drafts of your document.</p>
        </Card>
      </Layout.Section>
    </Layout>
  </Page>
)

export default App
