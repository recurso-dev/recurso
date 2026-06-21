import React from 'react';
import ComponentCreator from '@docusaurus/ComponentCreator';

export default [
  {
    path: '/',
    component: ComponentCreator('/', 'eea'),
    routes: [
      {
        path: '/',
        component: ComponentCreator('/', '164'),
        routes: [
          {
            path: '/',
            component: ComponentCreator('/', 'cbe'),
            routes: [
              {
                path: '/api-reference/customers',
                component: ComponentCreator('/api-reference/customers', '1f3'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/api-reference/invoices',
                component: ComponentCreator('/api-reference/invoices', '753'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/api-reference/plans',
                component: ComponentCreator('/api-reference/plans', '796'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/api-reference/subscriptions',
                component: ComponentCreator('/api-reference/subscriptions', 'e32'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/getting-started/installation',
                component: ComponentCreator('/getting-started/installation', '654'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/getting-started/quickstart',
                component: ComponentCreator('/getting-started/quickstart', 'e50'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/intro',
                component: ComponentCreator('/intro', '9fa'),
                exact: true,
                sidebar: "tutorialSidebar"
              }
            ]
          }
        ]
      }
    ]
  },
  {
    path: '*',
    component: ComponentCreator('*'),
  },
];
