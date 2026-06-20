describe("Admin Panel — Time Requests", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    // 1. Log in as user and create a time request
    cy.get('input[type="text"]').clear().type("user@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    cy.contains("button", "Request Time").click();
    cy.get('input[type="number"]').type("{selectall}{backspace}" + "150");
    cy.get('textarea[placeholder="Running VQE experiments for chemistry simulations..."]')
      .clear().type("Testing time requests");
    const alertStub = cy.stub();
    cy.on("window:alert", alertStub);
    
    cy.contains("button", "Submit Time Request").click();
    
    cy.wrap(alertStub).should("have.been.calledWithMatch", /submitted successfully/);

    cy.contains("button", "Profile Settings").click();
    cy.contains("button", "Sign Out").click();
    cy.get('[data-testid="login-modal"]').should("be.visible");

    cy.intercept("GET", "**/api/collections/qpu_time_requests/records*").as("getRequests");
    
    // 2. Log in as admin
    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').clear().type("admin@example.com");
    cy.get('input[type="password"]').clear().type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    cy.wait("@getRequests").then((interception) => {
      cy.writeFile("cypress_response.log", JSON.stringify(interception.response?.body));
    });

    cy.contains("button", "Admin Panel").click();
    cy.contains("h1", "Admin Panel").should("be.visible");

    cy.contains("button", "Time Requests").click();
  });

  it("approves a pending time request and status changes to approved", () => {
    // Approve the first pending request
    cy.get('[data-testid="time-request-row"]').first().as('requestRow');
    
    cy.get('@requestRow').within(() => {
        cy.get('button svg.lucide-check')
          .parent()
          .click();
      });

    // The row should now show "approved" badge and "Processed" action
    cy.get('@requestRow').within(() => {
        cy.contains("span", "approved").should("be.visible");
        cy.contains("span", "Processed").should("be.visible");
      });
  });

  it("rejects a pending time request and status changes to rejected", () => {
    // Handle the browser prompt for rejection reason
    cy.window().then((win) => {
      cy.stub(win, "prompt").returns("Insufficient justification");
    });

    // Reject the first pending request
    cy.get('[data-testid="time-request-row"]').first().as('requestRow');
    
    cy.get('@requestRow').within(() => {
        cy.get('button svg.lucide-x')
          .parent()
          .click();
      });

    // The row should now show "rejected" badge and "Processed" action
    cy.get('@requestRow').within(() => {
        cy.contains("span", "rejected").should("be.visible");
        cy.contains("span", "Processed").should("be.visible");
      });
  });
});
