import { createRootRoute, Link, Outlet } from '@tanstack/react-router';
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools';
import { Layout } from 'antd';
import { MessageProvider } from '../components/MessageProvider';
import { DeploymentStreamProvider } from '../context/DeploymentStreamContext';

const { Header, Content, Footer } = Layout;

export const Route = createRootRoute({
  component: RootComponent,
});

function RootComponent() {
  return (
    <MessageProvider>
      <DeploymentStreamProvider>
        <Layout style={{ minHeight: '100vh' }}>
          <Header style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div style={{ color: 'white', fontSize: '20px', fontWeight: 'bold' }}>
              <Link to="/" style={{ color: 'white', textDecoration: 'none' }}>Flink Admin</Link>
            </div>
          </Header>
          <Content style={{ padding: '24px', maxWidth: '90%', margin: '0 10%', flex: '1 0 auto' }}>
            <Outlet />
          </Content>
          <Footer style={{ textAlign: 'center' }}>
            Flink Admin {new Date().getFullYear()}
          </Footer>
          <TanStackRouterDevtools />
        </Layout>
      </DeploymentStreamProvider>
    </MessageProvider>
  );
}
