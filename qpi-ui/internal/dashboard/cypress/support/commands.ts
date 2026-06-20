/* eslint-disable @typescript-eslint/no-namespace */
/// <reference types="cypress" />

declare namespace Cypress {
  interface Chainable {
    resetDb(): Chainable<void>;
  }
}

Cypress.Commands.add("resetDb", () => {
  cy.request({
    method: "POST",
    url: "http://127.0.0.1:8090/api/collections/_superusers/auth-with-password",
    body: { identity: "admin@example.com", password: "supersecretpassword1234" },
    log: false,
  }).then((resp) => {
    const token = resp.body.token;
    const collectionsToClear = ["notifications", "qpu_time_requests"];

    collectionsToClear.forEach((col) => {
      cy.request({
        method: "GET",
        url: `http://127.0.0.1:8090/api/collections/${col}/records?perPage=500`,
        headers: { Authorization: token },
        log: false,
      }).then((res) => {
        const items = res.body.items || [];
        items.forEach((item: unknown) => {
          cy.request({
            method: "DELETE",
            url: `http://127.0.0.1:8090/api/collections/${col}/records/${item.id}`,
            headers: { Authorization: token },
            log: false,
          });
        });
      });
    });
  });
});
