import './commands';

// Catch unhandled exceptions to prevent the test from crashing on application errors
Cypress.on('uncaught:exception', () => {
  return false;
});

// Run DB reset before every test
beforeEach(() => {
  cy.resetDb();
});
