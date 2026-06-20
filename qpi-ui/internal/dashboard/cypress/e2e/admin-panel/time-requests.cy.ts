describe("Admin Panel — Time Requests", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').type("admin@example.com");
    cy.get('input[type="password"]').type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    cy.contains("button", "Admin Panel").click();
    cy.contains("h1", "Admin Panel").should("be.visible");

    cy.contains("button", "Time Requests").click();
    cy.contains("button", "Time Requests")
      .should("have.class", "border-b-2");
  });

  it("approves a pending time request and status changes to approved", () => {
    // Approve the first pending request
    cy.get('table tbody tr')
      .first()
      .within(() => {
        cy.get('button svg.lucide-check')
          .parent()
          .click();
      });

    // The row should now show "approved" badge and "Processed" action
    cy.get('table tbody tr')
      .first()
      .within(() => {
        cy.contains("span", "approved").should("be.visible");
        cy.contains("span", "Processed").should("be.visible");
      });
  });

  it("rejects a pending time request and status changes to rejected", () => {
    // Reject the first pending request
    cy.get('table tbody tr')
      .first()
      .within(() => {
        cy.get('button svg.lucide-x')
          .parent()
          .click();
      });

    // Handle the browser prompt for rejection reason
    cy.window().then((win) => {
      cy.stub(win, "prompt").returns("Insufficient justification");
    });

    // The row should now show "rejected" badge and "Processed" action
    cy.get('table tbody tr')
      .first()
      .within(() => {
        cy.contains("span", "rejected").should("be.visible");
        cy.contains("span", "Processed").should("be.visible");
      });
  });
});
