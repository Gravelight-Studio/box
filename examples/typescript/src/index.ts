import pino from 'pino';
import { createRouter, HandlerRegistry } from '@gravelight-studio/box';
import * as health from './handlers/health';
import * as users from './handlers/users';

async function main() {
  // Create logger
  const logger = pino({
    level: 'info',
    transport: {
      target: 'pino-pretty',
      options: {
        colorize: true
      }
    }
  });

  logger.info('Starting Box Example API');

  try {
    // Create Box router
    const router = await createRouter({
      handlersDir: __dirname + '/handlers',
      db: null, // In real app, initialize database connection here
      logger
    });

    // Create handler registry
    const registry = new HandlerRegistry(null, logger);

    // Register health handlers
    registry.register('health', 'getHealth', health.getHealth);

    // Register user handlers
    registry.register('users', 'listUsers', users.listUsers);
    registry.register('users', 'getUser', users.getUser);
    registry.register('users', 'createUser', users.createUser);
    registry.register('users', 'streamUserEvents', users.streamUserEvents);

    // Register all handlers with router
    router.registerHandlers(registry);

    // Log registered routes
    const handlers = router.getHandlers();
    logger.info(`Registered ${handlers.length} handlers`);

    for (const handler of handlers) {
      logger.info({
        method: handler.route.method,
        path: handler.route.path,
        deployment: handler.deploymentType,
        function: handler.functionName
      }, 'Route registered');
    }

    // Start server
    const port = process.env.PORT || 8080;
    const app = router.getApp();

    app.listen(port, () => {
      logger.info(`Server starting on :${port}`);
      logger.info(`URL: http://localhost:${port}`);
      logger.info('Example endpoints:');
      logger.info(`  Health: http://localhost:${port}/health`);
      logger.info(`  Users: http://localhost:${port}/api/v1/users`);
      logger.info(`  Stream: http://localhost:${port}/api/v1/users/123/events`);
    });
  } catch (error) {
    logger.error(error, 'Failed to start server');
    process.exit(1);
  }
}

main();
