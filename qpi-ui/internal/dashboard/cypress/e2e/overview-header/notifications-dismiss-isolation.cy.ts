describe("Overview & Header — Notification Dismiss Isolation", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");
  });

  it("still shows a notification to another user after one user dismisses it", () => {
    // 1. Admin broadcasts an announcement
    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').clear().type("admin@example.com");
    cy.get('input[type="password"]').clear().type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    cy.contains("button", "Admin Panel").click();
    cy.contains("h1", "Admin Panel").should("be.visible");
    cy.contains("button", "Broadcast Announcement").click();

    const uniqueTitle = `Shared Announcement`;

    cy.get('input[placeholder="QPU Maintenance Schedule"]')
      .clear().type(uniqueTitle);
    cy.get('textarea[placeholder="Rigetti Aspen-9 will be offline for calibration tomorrow..."]')
      .clear().type(`This is a shared broadcast announcement.`);
    
    // Fill datetime
    const pad = (n: number) => n.toString().padStart(2, "0");
    const start = new Date(Date.now() - 5 * 60000);
    const end = new Date(Date.now() + 20 * 60000);
    const startStr = `${start.getFullYear()}-${pad(start.getMonth() + 1)}-${pad(start.getDate())}T${pad(start.getHours())}:${pad(start.getMinutes())}`;
    const endStr = `${end.getFullYear()}-${pad(end.getMonth() + 1)}-${pad(end.getDate())}T${pad(end.getHours())}:${pad(end.getMinutes())}`;

    cy.get('input[type="datetime-local"]').first().type(startStr);
    cy.get('input[type="datetime-local"]').last().type(endStr);

    cy.get("form").contains("button", "Broadcast Announcement").click();

    cy.contains("Announcement broadcasted successfully!").should("be.visible");

    // 2. Log out and log in as user@example.com
    cy.contains("button", "Profile Settings").click();
    cy.contains("button", "Sign Out").click();
    cy.get('[data-testid="login-modal"]').should("be.visible");

    cy.get('input[type="text"]').clear().type("user@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");
    cy.visit("/#overview");

    // Verify the notification is visible
    cy.contains("h3", "System Announcements")
      .parent()
      .within(() => {
        cy.contains(uniqueTitle).scrollIntoView().should("be.visible");
      });

    // Dismiss the notification from the panel
    cy.contains("h3", "System Announcements")
      .parent()
      .within(() => {
        cy.contains(uniqueTitle)
          .parent()
          .parent()
          .parent()
          .trigger("mouseover")
          .within(() => {
            cy.get('button svg.lucide-x').parent().click({ force: true });
          });
      });

    // Verify it's gone for this user
    cy.contains("h3", "System Announcements")
      .parent()
      .within(() => {
        cy.contains(uniqueTitle).should("not.exist");
      });

    // 3. Log out and log in as admin again
    cy.contains("button", "Profile Settings").click();
    cy.contains("button", "Sign Out").click();
    cy.get('[data-testid="login-modal"]').should("be.visible");

    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').clear().type("admin@example.com");
    cy.get('input[type="password"]').clear().type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");
    cy.visit("/#overview");

    // The notification should still be visible to admin (not dismissed by them)
    cy.contains("h3", "System Announcements")
      .parent()
      .within(() => {
        cy.contains(uniqueTitle).scrollIntoView().should("be.visible");
      });
  });
});
