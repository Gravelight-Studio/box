import { Request, Response, HandlerFactory, Logger, Pool } from '@gravelight-studio/box';

interface User {
  id: string;
  email: string;
  name: string;
  createdAt: Date;
}

interface CreateUserRequest {
  email: string;
  name: string;
}

// @box:function
// @box:path GET /api/v1/users
// @box:auth required
// @box:ratelimit 100/minute
// @box:cors origins=*
export const listUsers: HandlerFactory = (db: Pool | null, logger: Logger) => {
  return (req: Request, res: Response) => {
    logger.info('Listing users');

    // Mock data
    const users: User[] = [
      {
        id: 'user-1',
        email: 'alice@example.com',
        name: 'Alice',
        createdAt: new Date(Date.now() - 24 * 60 * 60 * 1000)
      },
      {
        id: 'user-2',
        email: 'bob@example.com',
        name: 'Bob',
        createdAt: new Date(Date.now() - 12 * 60 * 60 * 1000)
      }
    ];

    res.status(200).json(users);
  };
};

// @box:function
// @box:path GET /api/v1/users/:id
// @box:auth required
// @box:ratelimit 200/minute
// @box:cors origins=*
export const getUser: HandlerFactory = (db: Pool | null, logger: Logger) => {
  return (req: Request, res: Response) => {
    const { id } = req.params;
    logger.info(`Getting user: ${id}`);

    // Mock data
    const user: User = {
      id,
      email: 'user@example.com',
      name: 'Example User',
      createdAt: new Date(Date.now() - 24 * 60 * 60 * 1000)
    };

    res.status(200).json(user);
  };
};

// @box:function
// @box:path POST /api/v1/users
// @box:auth required
// @box:ratelimit 50/hour
// @box:cors origins=*
// @box:timeout 10s
// @box:memory 256MB
export const createUser: HandlerFactory = (db: Pool | null, logger: Logger) => {
  return (req: Request, res: Response) => {
    const userData = req.body as CreateUserRequest;
    logger.info(`Creating user: ${userData.email}`);

    // Mock creation
    const user: User = {
      id: 'user-new',
      email: userData.email,
      name: userData.name,
      createdAt: new Date()
    };

    res.status(201).json(user);
  };
};

// @box:container
// @box:path GET /api/v1/users/:id/events
// @box:auth required
// @box:timeout 5m
// @box:concurrency 100
// @box:cors origins=*
export const streamUserEvents: HandlerFactory = (db: Pool | null, logger: Logger) => {
  return (req: Request, res: Response) => {
    const { id } = req.params;
    logger.info(`Streaming events for user: ${id}`);

    // Set headers for SSE
    res.setHeader('Content-Type', 'text/event-stream');
    res.setHeader('Cache-Control', 'no-cache');
    res.setHeader('Connection', 'keep-alive');

    // Send events
    let count = 0;
    const interval = setInterval(() => {
      if (count >= 5) {
        clearInterval(interval);
        res.end();
        logger.info('Event stream completed');
        return;
      }

      const event = {
        event: 'user_activity',
        user_id: id,
        timestamp: new Date(),
        data: 'User performed action'
      };

      res.write(`data: ${JSON.stringify(event)}\n\n`);
      count++;
      logger.info(`Sent event ${count}/5`);
    }, 2000);

    // Handle client disconnect
    req.on('close', () => {
      clearInterval(interval);
      logger.info('Client disconnected');
    });
  };
};
